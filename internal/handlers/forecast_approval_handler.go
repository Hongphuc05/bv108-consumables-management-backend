package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"

	"github.com/gin-gonic/gin"
)

type ForecastApprovalHandler struct {
	repo      *models.ForecastApprovalRepository
	userRepo  *models.UserRepository
	jwtSecret []byte
	hub       *realtime.Hub
}

type SaveForecastApprovalRequest struct {
	ForecastMonth int    `json:"forecastMonth" binding:"required"`
	ForecastYear  int    `json:"forecastYear" binding:"required"`
	MaQuanLy      string `json:"maQuanLy"`
	MaVtytCu      string `json:"maVtytCu" binding:"required"`
	TenVtytBv     string `json:"tenVtytBv" binding:"required"`
	Status        string `json:"status" binding:"required"`
	LyDo          string `json:"lyDo"`
	DuTruGoc      *int   `json:"duTruGoc"`
	DuTruSua      *int   `json:"duTruSua"`
}

type SaveForecastApprovalsRequest struct {
	Items []SaveForecastApprovalRequest `json:"items" binding:"required"`
}

func NewForecastApprovalHandler(repo *models.ForecastApprovalRepository, userRepo *models.UserRepository, jwtSecret string, hub *realtime.Hub) *ForecastApprovalHandler {
	return &ForecastApprovalHandler{
		repo:      repo,
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		hub:       hub,
	}
}

func (h *ForecastApprovalHandler) GetForecastApprovals(c *gin.Context) {
	if _, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	now := time.Now()
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))

	if month < 1 || month > 12 || year < 2000 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month/year is invalid"})
		return
	}

	records, err := h.repo.ListByMonthYear(month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *ForecastApprovalHandler) GetForecastChangeHistory(c *gin.Context) {
	if _, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	latestOnlyRaw := strings.TrimSpace(c.DefaultQuery("latestOnly", "0"))
	latestOnly := latestOnlyRaw == "1" || strings.EqualFold(latestOnlyRaw, "true")

	if month < 0 || month > 12 || year < 0 || (year > 0 && year < 2000) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month/year is invalid"})
		return
	}

	records, err := h.repo.ListChangeHistory(limit, month, year, latestOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *ForecastApprovalHandler) GetForecastMonthlyHistory(c *gin.Context) {
	if _, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	records, err := h.repo.ListMonthlyChangeHistory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *ForecastApprovalHandler) SaveForecastApproval(c *gin.Context) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req SaveForecastApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid forecast approval payload"})
		return
	}

	input, err := buildForecastApprovalInput(req, currentUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: err.Error()})
		return
	}

	if err := h.repo.SaveApproval(input); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	h.broadcastForecastApprovalUpdated(currentUser, input.ForecastMonth, input.ForecastYear, input.Status, 1)
	c.JSON(http.StatusOK, gin.H{"message": "Forecast approval saved successfully"})
}

func (h *ForecastApprovalHandler) SaveForecastApprovalsBulk(c *gin.Context) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req SaveForecastApprovalsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid forecast approvals payload"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one approval item is required"})
		return
	}

	inputs := make([]models.SaveForecastApprovalInput, 0, len(req.Items))
	for _, item := range req.Items {
		input, err := buildForecastApprovalInput(item, currentUser)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: err.Error()})
			return
		}
		inputs = append(inputs, input)
	}

	if err := h.repo.SaveApprovals(inputs); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	firstInput := inputs[0]
	h.broadcastForecastApprovalUpdated(currentUser, firstInput.ForecastMonth, firstInput.ForecastYear, firstInput.Status, len(inputs))

	c.JSON(http.StatusOK, gin.H{"message": "Forecast approvals saved successfully", "count": len(inputs)})
}

func (h *ForecastApprovalHandler) broadcastForecastApprovalUpdated(currentUser *models.UserProfile, month int, year int, status string, count int) {
	if h.hub == nil {
		return
	}

	normalizedStatus := strings.TrimSpace(status)
	h.hub.Broadcast("forecast.approvals_updated", gin.H{
		"month":     month,
		"year":      year,
		"status":    normalizedStatus,
		"count":     count,
		"updatedBy": currentUser.Username,
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func buildForecastApprovalInput(req SaveForecastApprovalRequest, currentUser *models.UserProfile) (models.SaveForecastApprovalInput, error) {
	status := strings.TrimSpace(req.Status)
	if status != models.ForecastApprovalStatusApproved && status != models.ForecastApprovalStatusRejected && status != models.ForecastApprovalStatusEdited {
		return models.SaveForecastApprovalInput{}, fmt.Errorf("status is invalid")
	}

	if req.ForecastMonth < 1 || req.ForecastMonth > 12 || req.ForecastYear < 2000 {
		return models.SaveForecastApprovalInput{}, fmt.Errorf("forecast month/year is invalid")
	}

	maVtytCu := strings.TrimSpace(req.MaVtytCu)
	tenVtytBv := strings.TrimSpace(req.TenVtytBv)
	if maVtytCu == "" || tenVtytBv == "" {
		return models.SaveForecastApprovalInput{}, fmt.Errorf("maVtytCu and tenVtytBv are required")
	}

	return models.SaveForecastApprovalInput{
		ForecastMonth: req.ForecastMonth,
		ForecastYear:  req.ForecastYear,
		MaQuanLy:      strings.TrimSpace(req.MaQuanLy),
		MaVtytCu:      maVtytCu,
		TenVtytBv:     tenVtytBv,
		Status:        status,
		LyDo:          strings.TrimSpace(req.LyDo),
		DuTruGoc:      req.DuTruGoc,
		DuTruSua:      req.DuTruSua,
		Reviewer: models.OrderActor{
			ID:       currentUser.ID,
			Username: currentUser.Username,
			Email:    currentUser.Email,
		},
		ReviewedAt: time.Now().Format(time.RFC3339),
	}, nil
}
