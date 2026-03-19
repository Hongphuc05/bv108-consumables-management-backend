package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/handlers"
	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"
	"bv108-consumables-management-backend/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
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
	if err := orderRepo.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize order history schema:", err)
	}
	orderUnreadRepo := models.NewOrderUnreadRepository(database.DB)
	if err := orderUnreadRepo.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize unread schema:", err)
	}
	companyContactRepo := models.NewCompanyContactRepository(database.DB)
	if err := companyContactRepo.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize company contacts schema:", err)
	}
	if err := companyContactRepo.SyncFromExistingData(models.DefaultCompanyContactEmail); err != nil {
		log.Fatal("Failed to sync company contacts:", err)
	}
	if err := companyContactRepo.BackfillOrderReferences(); err != nil {
		log.Fatal("Failed to backfill company contact relations:", err)
	}
	orderMailer := services.NewSMTPOrderMailer(
		config.AppConfig.SMTPHost,
		config.AppConfig.SMTPPort,
		config.AppConfig.SMTPUsername,
		config.AppConfig.SMTPAppPassword,
		config.AppConfig.SMTPFrom,
	)
	realtimeHub := realtime.NewHub()
	wsHandler := handlers.NewWSHandler(userRepo, config.AppConfig.JWTSecret, realtimeHub)
	orderHandler := handlers.NewOrderHandler(orderRepo, orderUnreadRepo, companyContactRepo, userRepo, config.AppConfig.JWTSecret, orderMailer, realtimeHub)
	forecastApprovalRepo := models.NewForecastApprovalRepository(database.DB)
	if err := forecastApprovalRepo.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize forecast approval schema:", err)
	}
	forecastApprovalHandler := handlers.NewForecastApprovalHandler(forecastApprovalRepo, userRepo, config.AppConfig.JWTSecret)

	hoaDonRepo := models.NewHoaDonRepository(database.DB)
	hoaDonHandler := handlers.NewHoaDonHandler(hoaDonRepo)
	refreshHandler := handlers.NewRefreshHandler(hoaDonRepo)

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
			supplies.GET("", supplyHandler.GetAllSupplies)                    // GET /api/supplies?page=1&pageSize=20
			supplies.GET("/search", supplyHandler.SearchSupplies)             // GET /api/supplies/search?keyword=xxx
			supplies.GET("/groups", supplyHandler.GetAllGroups)               // GET /api/supplies/groups
			supplies.GET("/group", supplyHandler.GetSuppliesByGroup)          // GET /api/supplies/group?groupName=xxx
			supplies.GET("/low-stock", supplyHandler.GetLowStockSupplies)     // GET /api/supplies/low-stock?threshold=20
			supplies.GET("/compare-groups", supplyHandler.GetCompareGroups)   // GET /api/supplies/compare-groups
			supplies.GET("/compare-catalog", supplyHandler.GetCompareCatalog) // GET /api/supplies/compare-catalog?page=1&pageSize=20&keyword=xxx&groupFilter=yyy
			supplies.POST("/compare", supplyHandler.CompareSupplies)          // POST /api/supplies/compare
			supplies.GET("/:id", supplyHandler.GetSupplyByID)                 // GET /api/supplies/:id
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
			orders.GET("/company-contacts/search", orderHandler.SearchCompanyContacts)
			orders.GET("/unread-snapshot", orderHandler.GetUnreadSnapshot)
			orders.POST("/pending/forecast", orderHandler.CreateForecastOrders)
			orders.POST("/pending/manual", orderHandler.CreateManualOrder)
			orders.POST("/place", orderHandler.PlaceOrders)
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

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
