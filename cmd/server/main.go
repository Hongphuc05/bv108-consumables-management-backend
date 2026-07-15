package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/handlers"
	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"
	"bv108-consumables-management-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	bootstrapStartedAt := time.Now()

	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	gin.SetMode(config.AppConfig.GinMode)

	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDB()

	supplyRepo := models.NewSupplyRepository(database.DB)
	userRepo := models.NewUserRepository(database.DB)
	supplyTaskRepo := models.NewSupplyTaskRepository(database.DB)
	orderRepo := models.NewOrderRepository(database.DB)
	invoiceMatchRepo := models.NewInvoiceReconciliationRepository(database.DB)
	orderUnreadRepo := models.NewOrderUnreadRepository(database.DB)
	companyContactRepo := models.NewCompanyContactRepository(database.DB)
	forecastApprovalRepo := models.NewForecastApprovalRepository(database.DB)
	schemaMaintenanceRepo := models.NewSchemaMaintenanceRepository(database.DB)
	hoaDonRepo := models.NewHoaDonRepository(database.DB)

	mustRunStartupStepsParallel(
		startupStep{name: "order history schema", run: orderRepo.EnsureSchema},
		startupStep{name: "invoice reconciliation schema", run: invoiceMatchRepo.EnsureSchema},
		startupStep{name: "order unread schema", run: orderUnreadRepo.EnsureSchema},
		startupStep{name: "forecast approval schema", run: forecastApprovalRepo.EnsureSchema},
		startupStep{name: "supply task schema", run: supplyTaskRepo.EnsureSchema},
	)
	mustRunStartupStep("invoice export schema", schemaMaintenanceRepo.EnsureInvoiceExportSchema)
	mustRunStartupStep("company contacts schema", companyContactRepo.EnsureSchema)
	mustRunStartupStep("relational schema", schemaMaintenanceRepo.EnsureRelationalIntegrity)

	realtimeHub := realtime.NewHub()
	orderMailer := services.NewSMTPOrderMailer(services.SMTPOrderMailerConfig{
		Host:        config.AppConfig.SMTPHost,
		Port:        config.AppConfig.SMTPPort,
		Username:    config.AppConfig.SMTPUsername,
		AppPassword: config.AppConfig.SMTPAppPassword,
		From:        config.AppConfig.SMTPFrom,
		TLSPolicy:   config.AppConfig.SMTPTLSPolicy,
	})
	internalSupplySyncService := services.NewInternalSupplySyncService(config.AppConfig, supplyRepo, companyContactRepo)
	geminiProxyService := services.NewGeminiProxyService(services.GeminiProxyConfig{
		APIKey:          config.AppConfig.GeminiAPIKey,
		Model:           config.AppConfig.GeminiModel,
		APIBaseURL:      config.AppConfig.GeminiAPIBaseURL,
		EnableWebSearch: config.AppConfig.GeminiWebSearch,
		MaxOutputTokens: config.AppConfig.GeminiMaxOutputTokens,
	})
	vinmesCatalogService := services.NewVinmesCatalogService(services.VinmesCatalogConfig{
		APIBaseURL:     config.AppConfig.VinmesAPIBaseURL,
		APIToken:       config.AppConfig.VinmesAPIToken,
		TimeoutSeconds: config.AppConfig.VinmesAPITimeoutSeconds,
	})

	router := newRouter(config.AppConfig.FrontendURL, apiHandlers{
		auth: handlers.NewAuthHandler(
			userRepo,
			config.AppConfig.JWTSecret,
			config.AppConfig.JWTExpiresHours,
			config.AppConfig.JWTExpiresMinutes,
		),
		supplies:           handlers.NewSupplyHandler(supplyRepo, userRepo, supplyTaskRepo, config.AppConfig.JWTSecret),
		supplyTasks:        handlers.NewSupplyTaskHandler(supplyRepo, supplyTaskRepo, userRepo, config.AppConfig.JWTSecret),
		invoices:           handlers.NewHoaDonHandler(hoaDonRepo, userRepo, config.AppConfig.JWTSecret),
		invoiceRefresh:     handlers.NewRefreshHandler(hoaDonRepo, userRepo, config.AppConfig.JWTSecret, realtimeHub),
		internalSupplySync: handlers.NewInternalSupplySyncHandler(internalSupplySyncService, userRepo, config.AppConfig.JWTSecret),
		orders:             handlers.NewOrderHandler(orderRepo, invoiceMatchRepo, orderUnreadRepo, companyContactRepo, userRepo, config.AppConfig.JWTSecret, orderMailer, realtimeHub, vinmesCatalogService),
		forecastApprovals:  handlers.NewForecastApprovalHandler(forecastApprovalRepo, userRepo, config.AppConfig.JWTSecret, realtimeHub),
		reports:            handlers.NewReportHandler(userRepo, config.AppConfig.JWTSecret, geminiProxyService),
		websocket:          handlers.NewWSHandler(userRepo, config.AppConfig.JWTSecret, realtimeHub, config.AppConfig.FrontendURL),
	})

	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()
	internalSupplySyncService.Start(backgroundCtx)

	go func() {
		if err := router.Run(":" + config.AppConfig.ServerPort); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	log.Printf("Server is running on http://localhost:%s", config.AppConfig.ServerPort)
	log.Printf("API documentation available at http://localhost:%s/health", config.AppConfig.ServerPort)
	log.Printf("[startup] bootstrap completed in %s", time.Since(bootstrapStartedAt).Round(time.Millisecond))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancelBackground()
}
