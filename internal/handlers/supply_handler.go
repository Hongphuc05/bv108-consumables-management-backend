package handlers

import (
	"log"
	"net/http"
	"strconv"

	"bv108-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type SupplyHandler struct {
	repo *models.SupplyRepository
}

// NewSupplyHandler creates a new supply handler
func NewSupplyHandler(repo *models.SupplyRepository) *SupplyHandler {
	return &SupplyHandler{repo: repo}
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	Total      int         `json:"total"`
	TotalPages int         `json:"totalPages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetAllSupplies godoc
// @Summary Get all supplies with pagination
// @Description Get all medical supplies from database with pagination support
// @Tags supplies
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} PaginationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies [get]
func (h *SupplyHandler) GetAllSupplies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	supplies, total, err := h.repo.GetAll(page, pageSize)
	if err != nil {
		log.Printf("❌ Error getting supplies: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, PaginationResponse{
		Data:       supplies,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSupplyByID godoc
// @Summary Get supply by ID
// @Description Get a specific medical supply by IDX1
// @Tags supplies
// @Param id path int true "Supply IDX1"
// @Success 200 {object} models.Supply
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/{id} [get]
func (h *SupplyHandler) GetSupplyByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "Invalid supply ID",
		})
		return
	}

	supply, err := h.repo.GetByID(id)
	if err != nil {
		if err.Error() == "supply not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "NOT_FOUND",
				Message: "Supply not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, supply)
}

// SearchSupplies godoc
// @Summary Search supplies
// @Description Search supplies by name or ID
// @Tags supplies
// @Param keyword query string true "Search keyword"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} PaginationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/search [get]
func (h *SupplyHandler) SearchSupplies(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MISSING_KEYWORD",
			Message: "Search keyword is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	supplies, total, err := h.repo.SearchByName(keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, PaginationResponse{
		Data:       supplies,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetSuppliesByGroup godoc
// @Summary Get supplies by group
// @Description Get all supplies belonging to a specific group
// @Tags supplies
// @Param groupName query string true "Group name"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} PaginationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/group [get]
func (h *SupplyHandler) GetSuppliesByGroup(c *gin.Context) {
	groupName := c.Query("groupName")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MISSING_GROUP",
			Message: "Group name is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	supplies, total, err := h.repo.GetByGroup(groupName, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, PaginationResponse{
		Data:       supplies,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetAllGroups godoc
// @Summary Get all groups
// @Description Get all unique group names
// @Tags supplies
// @Success 200 {array} string
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/groups [get]
func (h *SupplyHandler) GetAllGroups(c *gin.Context) {
	groups, err := h.repo.GetAllGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"total":  len(groups),
	})
}

// GetLowStockSupplies godoc
// @Summary Get low stock supplies
// @Description Get supplies with stock below threshold
// @Tags supplies
// @Param threshold query int false "Stock threshold" default(20)
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} PaginationResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/low-stock [get]
func (h *SupplyHandler) GetLowStockSupplies(c *gin.Context) {
	threshold, _ := strconv.Atoi(c.DefaultQuery("threshold", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	supplies, total, err := h.repo.GetLowStock(threshold, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, PaginationResponse{
		Data:       supplies,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "OK",
		"message": "BV108 Consumables API is running",
	})
}
