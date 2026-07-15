package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *OrderHandler) GetExportToVinmes(c *gin.Context) {
	if !h.authorizeVinmesExport(c) {
		return
	}

	filter, ok := parseVinmesExportFilter(c)
	if !ok {
		return
	}

	sources, err := h.invoiceMatchRepo.ListVinmesExportSources(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	masters := models.BuildVinmesExportMasters(sources)

	c.JSON(http.StatusOK, gin.H{
		"data":         masters,
		"count":        len(masters),
		"detailCount":  len(sources),
		"month":        filter.Month,
		"year":         filter.Year,
		"all":          filter.All,
		"materialCode": filter.MaterialCode,
	})
}

func (h *OrderHandler) GetExportToVinmesMappingPreview(c *gin.Context) {
	if !h.authorizeVinmesExport(c) {
		return
	}
	if h.vinmesCatalog == nil || !h.vinmesCatalog.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "VINMES_NOT_CONFIGURED", Message: "VINMES_API_BASE_URL is not configured"})
		return
	}

	filter, ok := parseVinmesExportFilter(c)
	if !ok {
		return
	}
	sources, err := h.invoiceMatchRepo.ListVinmesExportSources(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	masters := models.BuildVinmesExportMasters(sources)
	mapped, err := h.vinmesCatalog.BuildMappingPreview(c.Request.Context(), masters)
	if err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "VINMES_API_ERROR", Message: err.Error()})
		return
	}

	invalidCount := 0
	for _, item := range mapped {
		if len(item.ValidationErrors) > 0 {
			invalidCount++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":         mapped,
		"count":        len(mapped),
		"detailCount":  len(sources),
		"invalidCount": invalidCount,
		"month":        filter.Month,
		"year":         filter.Year,
		"all":          filter.All,
		"materialCode": filter.MaterialCode,
	})
}

func (h *OrderHandler) authorizeVinmesExport(c *gin.Context) bool {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return false
	}
	if !canViewInvoiceWorkflowRole(currentUser.Role) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "FORBIDDEN", Message: "Authenticated users with a valid operational role can export Vinmes reconciliation data"})
		return false
	}
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return false
	}
	return true
}

func parseVinmesExportFilter(c *gin.Context) (models.VinmesExportFilter, bool) {
	filter := models.VinmesExportFilter{
		Month:        int(time.Now().Month()),
		Year:         time.Now().Year(),
		MaterialCode: strings.TrimSpace(c.Query("materialCode")),
		Limit:        200,
	}

	if rawAll := strings.TrimSpace(c.Query("all")); rawAll != "" {
		parsed, err := strconv.ParseBool(rawAll)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "all must be true or false"})
			return filter, false
		}
		filter.All = parsed
	}
	if rawMonth := strings.TrimSpace(c.Query("month")); rawMonth != "" {
		parsed, err := strconv.Atoi(rawMonth)
		if err != nil || parsed < 1 || parsed > 12 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month must be from 1 to 12"})
			return filter, false
		}
		filter.Month = parsed
	}
	if rawYear := strings.TrimSpace(c.Query("year")); rawYear != "" {
		parsed, err := strconv.Atoi(rawYear)
		if err != nil || parsed < 2000 || parsed > 3000 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "year is invalid"})
			return filter, false
		}
		filter.Year = parsed
	}
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "limit must be greater than 0"})
			return filter, false
		}
		filter.Limit = parsed
	}

	return filter, true
}
