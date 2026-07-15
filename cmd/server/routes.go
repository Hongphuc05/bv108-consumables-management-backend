package main

import (
	"bv108-consumables-management-backend/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type apiHandlers struct {
	auth               *handlers.AuthHandler
	supplies           *handlers.SupplyHandler
	supplyTasks        *handlers.SupplyTaskHandler
	invoices           *handlers.HoaDonHandler
	invoiceRefresh     *handlers.RefreshHandler
	internalSupplySync *handlers.InternalSupplySyncHandler
	orders             *handlers.OrderHandler
	forecastApprovals  *handlers.ForecastApprovalHandler
	reports            *handlers.ReportHandler
	websocket          *handlers.WSHandler
}

func newRouter(frontendURL string, h apiHandlers) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(corsConfig(frontendURL)))
	router.GET("/health", handlers.HealthCheck)

	registerAPIRoutes(router.Group("/api"), h)
	return router
}

func corsConfig(frontendURL string) cors.Config {
	return cors.Config{
		AllowOrigins:     []string{frontendURL, "http://localhost:5173", "http://localhost:5174", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
}

func registerAPIRoutes(api *gin.RouterGroup, h apiHandlers) {
	api.GET("/ws", h.websocket.Handle)
	api.GET("/export-to-vinmes", h.orders.GetExportToVinmes)

	registerAuthRoutes(api.Group("/auth"), h.auth)
	registerSupplyRoutes(api.Group("/supplies"), h.supplies, h.internalSupplySync)
	registerSupplyTaskRoutes(api.Group("/supply-tasks"), h.supplyTasks)
	registerInvoiceRoutes(api.Group("/hoa-don"), h.invoices, h.invoiceRefresh)
	registerOrderRoutes(api.Group("/orders"), h.orders)
	registerForecastApprovalRoutes(api.Group("/forecast-approvals"), h.forecastApprovals)
	api.POST("/reports/gemini-compare", h.reports.GenerateGeminiCompare)
}

func registerAuthRoutes(group *gin.RouterGroup, h *handlers.AuthHandler) {
	group.POST("/register", h.Register)
	group.POST("/login", h.Login)
	group.GET("/profile", h.GetProfile)
	group.PUT("/profile", h.UpdateProfile)
	group.GET("/users", h.ListManagedUsers)
	group.PUT("/users/:id/role", h.UpdateManagedUserRole)
	group.PUT("/users/:id/password", h.ResetManagedUserPassword)
	group.DELETE("/users/:id", h.DeleteManagedUser)
}

func registerSupplyRoutes(group *gin.RouterGroup, h *handlers.SupplyHandler, syncHandler *handlers.InternalSupplySyncHandler) {
	group.GET("", h.GetAllSupplies)
	group.GET("/search", h.SearchSupplies)
	group.GET("/groups", h.GetAllGroups)
	group.GET("/group", h.GetSuppliesByGroup)
	group.GET("/low-stock", h.GetLowStockSupplies)
	group.GET("/compare-level1", h.GetCompareLevel1Options)
	group.GET("/compare-level2", h.GetCompareLevel2Options)
	group.GET("/compare-catalog", h.GetCompareCatalog)
	group.GET("/compare-export", h.ExportCompareCatalogExcel)
	group.POST("/compare-import", h.ImportCompareCatalogExcel)
	group.GET("/forecast-catalog", h.GetForecastCatalog)
	group.POST("/internal-sync", syncHandler.SyncNow)
	group.POST("/compare", h.CompareSupplies)
	group.GET("/:id", h.GetSupplyByID)
}

func registerSupplyTaskRoutes(group *gin.RouterGroup, h *handlers.SupplyTaskHandler) {
	group.GET("/state", h.GetState)
	group.GET("/catalog", h.GetSupplyCatalog)
	group.GET("/assignments", h.GetAssignmentsByUser)
	group.GET("/assignments/export", h.ExportAssignments)
	group.POST("/assignments/import", h.ImportAssignments)
	group.PUT("/visibility", h.UpdateVisibility)
	group.PUT("/assignments", h.UpdateAssignmentsByUser)
}

func registerInvoiceRoutes(group *gin.RouterGroup, h *handlers.HoaDonHandler, refreshHandler *handlers.RefreshHandler) {
	group.GET("", h.GetAllHoaDon)
	group.GET("/search", h.SearchHoaDon)
	group.GET("/:id", h.GetHoaDonByID)
	group.POST("/refresh", refreshHandler.RefreshInvoices)
}

func registerOrderRoutes(group *gin.RouterGroup, h *handlers.OrderHandler) {
	group.GET("/pending", h.GetPendingOrders)
	group.GET("/history", h.GetOrderHistory)
	group.GET("/invoice-reconciliations", h.GetInvoiceReconciliationHistory)
	group.GET("/invoice-reconciliations/matched-invoices", h.GetMatchedInvoiceNumbers)
	group.GET("/invoice-reconciliations/matched-orders", h.GetMatchedOrderReconciliations)
	group.GET("/company-contacts/search", h.SearchCompanyContacts)
	group.GET("/unread-snapshot", h.GetUnreadSnapshot)
	group.POST("/pending/forecast", h.CreateForecastOrders)
	group.POST("/pending/manual", h.CreateManualOrder)
	group.POST("/place", h.PlaceOrders)
	group.POST("/history/reorder", h.RepeatOrderHistory)
	group.POST("/invoice-reconciliations/upsert", h.UpsertInvoiceReconciliations)
	group.POST("/invoice-reconciliations/bulk", h.SaveInvoiceReconciliations)
	group.POST("/alerts/suppliers/seen", h.MarkSupplierAlertSeen)
	group.POST("/groups/seen", h.MarkGroupsSeen)
}

func registerForecastApprovalRoutes(group *gin.RouterGroup, h *handlers.ForecastApprovalHandler) {
	group.GET("", h.GetForecastApprovals)
	group.GET("/history", h.GetForecastChangeHistory)
	group.GET("/monthly-history", h.GetForecastMonthlyHistory)
	group.POST("", h.SaveForecastApproval)
	group.POST("/bulk", h.SaveForecastApprovalsBulk)
}
