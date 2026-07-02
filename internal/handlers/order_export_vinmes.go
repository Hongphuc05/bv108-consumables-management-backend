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
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return
	}

	all := false
	if rawAll := strings.TrimSpace(c.Query("all")); rawAll != "" {
		parsed, parseErr := strconv.ParseBool(rawAll)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "all must be true or false"})
			return
		}
		all = parsed
	}

	month := int(time.Now().Month())
	if rawMonth := strings.TrimSpace(c.Query("month")); rawMonth != "" {
		parsed, parseErr := strconv.Atoi(rawMonth)
		if parseErr != nil || parsed < 1 || parsed > 12 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month must be from 1 to 12"})
			return
		}
		month = parsed
	}

	year := time.Now().Year()
	if rawYear := strings.TrimSpace(c.Query("year")); rawYear != "" {
		parsed, parseErr := strconv.Atoi(rawYear)
		if parseErr != nil || parsed < 2000 || parsed > 3000 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "year is invalid"})
			return
		}
		year = parsed
	}

	limit := 200
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "limit must be greater than 0"})
			return
		}
		limit = parsed
	}

	filter := models.VinmesExportFilter{
		Month:        month,
		Year:         year,
		All:          all,
		MaterialCode: strings.TrimSpace(c.Query("materialCode")),
		Limit:        limit,
	}

	sources, err := h.invoiceMatchRepo.ListVinmesExportSources(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	items := make([]models.VinmesExportItem, 0, len(sources))
	for _, source := range sources {
		items = append(items, models.BuildVinmesExportItem(source))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         items,
		"count":        len(items),
		"month":        month,
		"year":         year,
		"all":          all,
		"materialCode": filter.MaterialCode,
	})
}
