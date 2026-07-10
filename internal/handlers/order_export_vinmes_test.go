package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetExportToVinmesRequiresAuthentication(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/export-to-vinmes", nil)

	handler := &OrderHandler{}
	handler.GetExportToVinmes(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GetExportToVinmes() status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
