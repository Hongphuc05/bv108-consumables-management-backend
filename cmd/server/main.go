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
		supplies := api.Group("/supplies")
		{
			supplies.GET("", supplyHandler.GetAllSupplies)                // GET /api/supplies?page=1&pageSize=20
			supplies.GET("/search", supplyHandler.SearchSupplies)         // GET /api/supplies/search?keyword=xxx
			supplies.GET("/groups", supplyHandler.GetAllGroups)           // GET /api/supplies/groups
			supplies.GET("/group", supplyHandler.GetSuppliesByGroup)      // GET /api/supplies/group?groupName=xxx
			supplies.GET("/low-stock", supplyHandler.GetLowStockSupplies) // GET /api/supplies/low-stock?threshold=20
			supplies.GET("/:id", supplyHandler.GetSupplyByID)             // GET /api/supplies/:id
		}

		hoaDon := api.Group("/hoa-don")
		{
			hoaDon.GET("", hoaDonHandler.GetAllHoaDon)              // GET /api/hoa-don?limit=100&offset=0
			hoaDon.GET("/search", hoaDonHandler.SearchHoaDon)       // GET /api/hoa-don/search?q=keyword
			hoaDon.GET("/:id", hoaDonHandler.GetHoaDonByID)         // GET /api/hoa-don/:id
			hoaDon.POST("/refresh", refreshHandler.RefreshInvoices) // POST /api/hoa-don/refresh
		}
	}

	// Graceful shutdown
	go func() {
		if err := router.Run(":" + config.AppConfig.ServerPort); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	log.Printf("🚀 Server is running on http://localhost:%s", config.AppConfig.ServerPort)
	log.Printf("📊 API documentation available at http://localhost:%s/health", config.AppConfig.ServerPort)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
