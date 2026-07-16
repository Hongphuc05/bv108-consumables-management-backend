package main

import (
	"testing"

	"bv108-consumables-management-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func TestRouterRegistersExpectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter("http://localhost:5173", apiHandlers{
		auth:               &handlers.AuthHandler{},
		supplies:           &handlers.SupplyHandler{},
		supplyTasks:        &handlers.SupplyTaskHandler{},
		invoices:           &handlers.HoaDonHandler{},
		invoiceRefresh:     &handlers.RefreshHandler{},
		internalSupplySync: &handlers.InternalSupplySyncHandler{},
		orders:             &handlers.OrderHandler{},
		forecastApprovals:  &handlers.ForecastApprovalHandler{},
		reports:            &handlers.ReportHandler{},
		websocket:          &handlers.WSHandler{},
	})

	expected := []string{
		"GET /health",
		"GET /api/ws",
		"GET /api/export-to-vinmes",
		"GET /api/export-to-vinmes/mapping-preview",
		"POST /api/export-to-vinmes/catalogs/refresh",
		"POST /api/auth/register",
		"POST /api/auth/login",
		"GET /api/auth/profile",
		"PUT /api/auth/profile",
		"GET /api/auth/users",
		"PUT /api/auth/users/:id/role",
		"PUT /api/auth/users/:id/password",
		"DELETE /api/auth/users/:id",
		"GET /api/supplies",
		"GET /api/supplies/search",
		"GET /api/supplies/groups",
		"GET /api/supplies/group",
		"GET /api/supplies/low-stock",
		"GET /api/supplies/compare-level1",
		"GET /api/supplies/compare-level2",
		"GET /api/supplies/compare-catalog",
		"GET /api/supplies/compare-export",
		"POST /api/supplies/compare-import",
		"GET /api/supplies/forecast-catalog",
		"POST /api/supplies/internal-sync",
		"POST /api/supplies/compare",
		"GET /api/supplies/:id",
		"GET /api/supply-tasks/state",
		"GET /api/supply-tasks/catalog",
		"GET /api/supply-tasks/assignments",
		"GET /api/supply-tasks/assignments/export",
		"POST /api/supply-tasks/assignments/import",
		"PUT /api/supply-tasks/visibility",
		"PUT /api/supply-tasks/assignments",
		"GET /api/hoa-don",
		"GET /api/hoa-don/search",
		"GET /api/hoa-don/:id",
		"POST /api/hoa-don/refresh",
		"GET /api/orders/pending",
		"GET /api/orders/history",
		"GET /api/orders/invoice-reconciliations",
		"GET /api/orders/invoice-reconciliations/matched-invoices",
		"GET /api/orders/invoice-reconciliations/matched-orders",
		"GET /api/orders/company-contacts/search",
		"GET /api/orders/unread-snapshot",
		"POST /api/orders/pending/forecast",
		"POST /api/orders/pending/manual",
		"POST /api/orders/place",
		"POST /api/orders/history/reorder",
		"POST /api/orders/invoice-reconciliations/upsert",
		"POST /api/orders/invoice-reconciliations/bulk",
		"POST /api/orders/alerts/suppliers/seen",
		"POST /api/orders/groups/seen",
		"GET /api/forecast-approvals",
		"GET /api/forecast-approvals/history",
		"GET /api/forecast-approvals/monthly-history",
		"POST /api/forecast-approvals",
		"POST /api/forecast-approvals/bulk",
		"POST /api/reports/gemini-compare",
	}

	actual := make(map[string]struct{}, len(router.Routes()))
	for _, route := range router.Routes() {
		actual[route.Method+" "+route.Path] = struct{}{}
	}

	if len(actual) != len(expected) {
		t.Fatalf("registered route count = %d, want %d", len(actual), len(expected))
	}
	for _, route := range expected {
		if _, ok := actual[route]; !ok {
			t.Errorf("missing route %s", route)
		}
	}
}
