package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/handlers"
	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"
	"bv108-consumables-management-backend/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type startupStep struct {
	name string
	run  func() error
}

func mustRunStartupStep(stepName string, fn func() error) {
	startedAt := time.Now()
	if err := fn(); err != nil {
		log.Fatalf("%s failed: %v", stepName, err)
	}
	log.Printf("[startup] %s completed in %s", stepName, time.Since(startedAt).Round(time.Millisecond))
}

func mustRunStartupStepsParallel(steps ...startupStep) {
	if len(steps) == 0 {
		return
	}

	type startupError struct {
		name string
		err  error
	}

	var wg sync.WaitGroup
	errCh := make(chan startupError, len(steps))

	for _, step := range steps {
		step := step
		wg.Add(1)

		go func() {
			defer wg.Done()

			startedAt := time.Now()
			if err := step.run(); err != nil {
				errCh <- startupError{name: step.name, err: err}
				return
			}

			log.Printf("[startup] %s completed in %s", step.name, time.Since(startedAt).Round(time.Millisecond))
		}()
	}

	wg.Wait()
	close(errCh)

	if startupErr, ok := <-errCh; ok {
		log.Fatalf("%s failed: %v", startupErr.name, startupErr.err)
	}
}

func runCompanyContactWarmup(repo *models.CompanyContactRepository) {
	startedAt := time.Now()
	log.Println("[startup] company contact warmup started (background)")

	if err := repo.SyncFromExistingData(models.DefaultCompanyContactEmail); err != nil {
		log.Printf("[startup] company contact warmup sync failed: %v", err)
		return
	}

	if err := repo.BackfillOrderReferences(); err != nil {
		log.Printf("[startup] company contact warmup backfill failed: %v", err)
		return
	}

	log.Printf("[startup] company contact warmup completed in %s", time.Since(startedAt).Round(time.Millisecond))
}

func main() {
	bootstrapStartedAt := time.Now()

	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set Gin mode
	gin.SetMode(config.AppConfig.GinMode)

	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDB()

	// Initialize repository and handler
	supplyRepo := models.NewSupplyRepository(database.DB)
	supplyHandler := handlers.NewSupplyHandler(supplyRepo)
	userRepo := models.NewUserRepository(database.DB)
	authHandler := handlers.NewAuthHandler(userRepo, config.AppConfig.JWTSecret, config.AppConfig.JWTExpiresHours)
	orderRepo := models.NewOrderRepository(database.DB)
	invoiceMatchRepo := models.NewInvoiceReconciliationRepository(database.DB)
	orderUnreadRepo := models.NewOrderUnreadRepository(database.DB)
	companyContactRepo := models.NewCompanyContactRepository(database.DB)
	forecastApprovalRepo := models.NewForecastApprovalRepository(database.DB)

	mustRunStartupStepsParallel(
		startupStep{name: "order history schema", run: orderRepo.EnsureSchema},
		startupStep{name: "invoice reconciliation schema", run: invoiceMatchRepo.EnsureSchema},
		startupStep{name: "order unread schema", run: orderUnreadRepo.EnsureSchema},
		startupStep{name: "forecast approval schema", run: forecastApprovalRepo.EnsureSchema},
	)
	mustRunStartupStep("company contacts schema", companyContactRepo.EnsureSchema)

	orderMailer := services.NewSMTPOrderMailer(
		config.AppConfig.SMTPHost,
		config.AppConfig.SMTPPort,
		config.AppConfig.SMTPUsername,
		config.AppConfig.SMTPAppPassword,
		config.AppConfig.SMTPFrom,
	)
	realtimeHub := realtime.NewHub()
	wsHandler := handlers.NewWSHandler(userRepo, config.AppConfig.JWTSecret, realtimeHub)
	orderHandler := handlers.NewOrderHandler(orderRepo, invoiceMatchRepo, orderUnreadRepo, companyContactRepo, userRepo, config.AppConfig.JWTSecret, orderMailer, realtimeHub)
	forecastApprovalHandler := handlers.NewForecastApprovalHandler(forecastApprovalRepo, userRepo, config.AppConfig.JWTSecret, realtimeHub)

	hoaDonRepo := models.NewHoaDonRepository(database.DB)
	hoaDonHandler := handlers.NewHoaDonHandler(hoaDonRepo)
	refreshHandler := handlers.NewRefreshHandler(hoaDonRepo, realtimeHub)

	// Initialize Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.AppConfig.FrontendURL, "http://localhost:5173", "http://localhost:5174", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check route
	router.GET("/health", handlers.HealthCheck)

	// API routes
	api := router.Group("/api")
	{
		api.GET("/ws", wsHandler.Handle)

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/profile", authHandler.GetProfile)
			auth.PUT("/profile", authHandler.UpdateProfile)
		}

		supplies := api.Group("/supplies")
		{
			supplies.GET("", supplyHandler.GetAllSupplies)                         // GET /api/supplies?page=1&pageSize=20
			supplies.GET("/search", supplyHandler.SearchSupplies)                  // GET /api/supplies/search?keyword=xxx
			supplies.GET("/groups", supplyHandler.GetAllGroups)                    // GET /api/supplies/groups
			supplies.GET("/group", supplyHandler.GetSuppliesByGroup)               // GET /api/supplies/group?groupName=xxx
			supplies.GET("/low-stock", supplyHandler.GetLowStockSupplies)          // GET /api/supplies/low-stock?threshold=20
			supplies.GET("/compare-level1", supplyHandler.GetCompareLevel1Options) // GET /api/supplies/compare-level1
			supplies.GET("/compare-level2", supplyHandler.GetCompareLevel2Options) // GET /api/supplies/compare-level2?level1=xxx
			supplies.GET("/compare-catalog", supplyHandler.GetCompareCatalog)      // GET /api/supplies/compare-catalog?page=1&pageSize=20&keyword=xxx&level1Filter=xxx&level2Filter=yyy
			supplies.POST("/compare", supplyHandler.CompareSupplies)               // POST /api/supplies/compare
			supplies.GET("/:id", supplyHandler.GetSupplyByID)                      // GET /api/supplies/:id
		}

		hoaDon := api.Group("/hoa-don")
		{
			hoaDon.GET("", hoaDonHandler.GetAllHoaDon)              // GET /api/hoa-don?limit=100&offset=0
			hoaDon.GET("/search", hoaDonHandler.SearchHoaDon)       // GET /api/hoa-don/search?q=keyword
			hoaDon.GET("/:id", hoaDonHandler.GetHoaDonByID)         // GET /api/hoa-don/:id
			hoaDon.POST("/refresh", refreshHandler.RefreshInvoices) // POST /api/hoa-don/refresh
		}

		orders := api.Group("/orders")
		{
			orders.GET("/pending", orderHandler.GetPendingOrders)
			orders.GET("/history", orderHandler.GetOrderHistory)
			orders.GET("/invoice-reconciliations", orderHandler.GetInvoiceReconciliationHistory)
			orders.GET("/invoice-reconciliations/matched-invoices", orderHandler.GetMatchedInvoiceNumbers)
			orders.GET("/invoice-reconciliations/matched-orders", orderHandler.GetMatchedOrderReconciliations)
			orders.GET("/company-contacts/search", orderHandler.SearchCompanyContacts)
			orders.GET("/unread-snapshot", orderHandler.GetUnreadSnapshot)
			orders.POST("/pending/forecast", orderHandler.CreateForecastOrders)
			orders.POST("/pending/manual", orderHandler.CreateManualOrder)
			orders.POST("/place", orderHandler.PlaceOrders)
			orders.POST("/invoice-reconciliations/bulk", orderHandler.SaveInvoiceReconciliations)
			orders.POST("/alerts/suppliers/seen", orderHandler.MarkSupplierAlertSeen)
			orders.POST("/groups/seen", orderHandler.MarkGroupsSeen)
		}

		forecastApprovals := api.Group("/forecast-approvals")
		{
			forecastApprovals.GET("", forecastApprovalHandler.GetForecastApprovals)
			forecastApprovals.GET("/history", forecastApprovalHandler.GetForecastChangeHistory)
			forecastApprovals.GET("/monthly-history", forecastApprovalHandler.GetForecastMonthlyHistory)
			forecastApprovals.POST("", forecastApprovalHandler.SaveForecastApproval)
			forecastApprovals.POST("/bulk", forecastApprovalHandler.SaveForecastApprovalsBulk)
		}
	}

	// Graceful shutdown
	go func() {
		if err := router.Run(":" + config.AppConfig.ServerPort); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	log.Printf("Server is running on http://localhost:%s", config.AppConfig.ServerPort)
	log.Printf("API documentation available at http://localhost:%s/health", config.AppConfig.ServerPort)
	log.Printf("[startup] bootstrap completed in %s", time.Since(bootstrapStartedAt).Round(time.Millisecond))
	go runCompanyContactWarmup(companyContactRepo)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
