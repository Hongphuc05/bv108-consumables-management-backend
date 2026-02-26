package handlers

import (
	"bv108-consumables-management-backend/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// HoaDonHandler handles HTTP requests for hoa_don
type HoaDonHandler struct {
	repo *models.HoaDonRepository
}

// NewHoaDonHandler creates a new handler instance
func NewHoaDonHandler(repo *models.HoaDonRepository) *HoaDonHandler {
	return &HoaDonHandler{repo: repo}
}

// GetAllHoaDon handles GET /api/hoa-don
func (h *HoaDonHandler) GetAllHoaDon(c *gin.Context) {
	// Pagination parameters
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get data
	hoaDons, err := h.repo.GetAll(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch invoices",
			"message": err.Error(),
		})
		return
	}

	// Get total count
	total, err := h.repo.GetCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get total count",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   hoaDons,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetHoaDonByID handles GET /api/hoa-don/:id
func (h *HoaDonHandler) GetHoaDonByID(c *gin.Context) {
	idHoaDon := c.Param("id")

	if idHoaDon == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invoice ID is required",
		})
		return
	}

	hoaDons, err := h.repo.GetByIDHoaDon(idHoaDon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch invoice",
			"message": err.Error(),
		})
		return
	}

	if len(hoaDons) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invoice not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

// SearchHoaDon handles GET /api/hoa-don/search
func (h *HoaDonHandler) SearchHoaDon(c *gin.Context) {
	keyword := c.Query("q")

	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search keyword is required",
		})
		return
	}

	// Pagination parameters
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	hoaDons, err := h.repo.SearchByKeyword(keyword, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search invoices",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    hoaDons,
		"keyword": keyword,
		"limit":   limit,
		"offset":  offset,
	})
}
