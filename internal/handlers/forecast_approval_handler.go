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

type forecastTransitionError struct {
	status  int
	message string
}

func (e *forecastTransitionError) Error() string {
	return e.message
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

	// Validate that the month/year is not in the past
	if err := validateForecastMonthNotInPast(req.ForecastMonth, req.ForecastYear); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "FORBIDDEN", Message: err.Error()})
		return
	}

	statusByItemKey, err := h.getForecastStatusByPeriod(req.ForecastMonth, req.ForecastYear)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	if err := validateForecastApprovalTransition(req, currentUser, lookupExistingForecastStatus(req, statusByItemKey)); err != nil {
		writeForecastTransitionError(c, err)
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

	// Validate that all month/years are not in the past
	for _, item := range req.Items {
		if err := validateForecastMonthNotInPast(item.ForecastMonth, item.ForecastYear); err != nil {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "FORBIDDEN", Message: err.Error()})
			return
		}
	}

	statusCacheByPeriod := make(map[string]map[string]string)
	inputs := make([]models.SaveForecastApprovalInput, 0, len(req.Items))
	for _, item := range req.Items {
		periodKey := fmt.Sprintf("%04d-%02d", item.ForecastYear, item.ForecastMonth)
		statusByItemKey, exists := statusCacheByPeriod[periodKey]
		if !exists {
			loaded, err := h.getForecastStatusByPeriod(item.ForecastMonth, item.ForecastYear)
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
				return
			}
			statusByItemKey = loaded
			statusCacheByPeriod[periodKey] = statusByItemKey
		}

		if err := validateForecastApprovalTransition(item, currentUser, lookupExistingForecastStatus(item, statusByItemKey)); err != nil {
			writeForecastTransitionError(c, err)
			return
		}

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
	now := time.Now().UTC()
	h.hub.Broadcast("forecast.approvals_updated", gin.H{
		"month":     month,
		"year":      year,
		"status":    normalizedStatus,
		"count":     count,
		"updatedBy": currentUser.Username,
		"updatedAt": now.Format(time.RFC3339Nano),
	})

	action := ""
	switch normalizedStatus {
	case models.ForecastApprovalStatusEdited:
		if normalizeRoleForPermissions(currentUser.Role) == RoleThuKho {
			action = "forecast.unsubmitted"
		} else {
			action = "forecast.edited"
		}
	case models.ForecastApprovalStatusSubmitted:
		action = "forecast.submitted"
	case models.ForecastApprovalStatusApproved:
		action = "forecast.approved"
	case models.ForecastApprovalStatusRejected:
		action = "forecast.rejected"
	}
	if action == "" {
		return
	}

	broadcastActivityNotification(h.hub, ActivityNotificationPayload{
		Category:   "forecast",
		Action:     action,
		ActorID:    currentUser.ID,
		ActorName:  currentUser.Username,
		ActorEmail: currentUser.Email,
		Count:      count,
		Status:     normalizedStatus,
		Month:      month,
		Year:       year,
		CreatedAt:  now.Format(time.RFC3339Nano),
	})
}

func buildForecastApprovalInput(req SaveForecastApprovalRequest, currentUser *models.UserProfile) (models.SaveForecastApprovalInput, error) {
	status := strings.TrimSpace(req.Status)
	if status != models.ForecastApprovalStatusApproved && status != models.ForecastApprovalStatusRejected && status != models.ForecastApprovalStatusEdited && status != models.ForecastApprovalStatusSubmitted {
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

func (h *ForecastApprovalHandler) getForecastStatusByPeriod(month, year int) (map[string]string, error) {
	records, err := h.repo.ListByMonthYear(month, year)
	if err != nil {
		return nil, err
	}

	statusByItemKey := make(map[string]string, len(records)*2)
	for _, record := range records {
		itemKey := forecastApprovalStatusKey(record.MaQuanLy, record.MaVtytCu)
		if itemKey != "" {
			statusByItemKey[itemKey] = strings.TrimSpace(record.Status)
		}

		fallbackKey := strings.TrimSpace(record.MaVtytCu)
		if fallbackKey != "" {
			statusByItemKey[fallbackKey] = strings.TrimSpace(record.Status)
		}
	}

	return statusByItemKey, nil
}

func writeForecastTransitionError(c *gin.Context, err error) {
	if transitionErr, ok := err.(*forecastTransitionError); ok {
		errorCode := "INVALID_REQUEST"
		if transitionErr.status == http.StatusForbidden {
			errorCode = "FORBIDDEN"
		}

		c.JSON(transitionErr.status, ErrorResponse{Error: errorCode, Message: transitionErr.message})
		return
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: err.Error()})
}

func validateForecastApprovalTransition(req SaveForecastApprovalRequest, currentUser *models.UserProfile, existingStatus string) error {
	normalizedStatus := strings.TrimSpace(req.Status)
	normalizedExistingStatus := strings.TrimSpace(existingStatus)

	isAdmin := userHasAnyRole(currentUser, RoleAdmin)

	switch normalizedStatus {
	case models.ForecastApprovalStatusEdited:
		if userHasAnyRole(currentUser, RoleThuKho) && !isAdmin {
			if normalizedExistingStatus != models.ForecastApprovalStatusSubmitted {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Only submitted forecasts can be unsubmitted by Thu kho"}
			}
			if req.DuTruGoc != nil || req.DuTruSua != nil || strings.TrimSpace(req.LyDo) != "" {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Thu kho unsubmit must not modify forecast values"}
			}
			return nil
		}

		if userHasAnyRole(currentUser, RoleNhanVienThau) || isAdmin {
			if normalizedExistingStatus == models.ForecastApprovalStatusSubmitted {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Forecast is submitted to Chi huy khoa. Thu kho must unsubmit first"}
			}
			if normalizedExistingStatus == models.ForecastApprovalStatusApproved {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Approved forecast cannot be edited"}
			}
			return nil
		}

		return &forecastTransitionError{status: http.StatusForbidden, message: "Only Nhan vien thau (or Admin) can edit forecasts"}

	case models.ForecastApprovalStatusSubmitted:
		if !(userHasAnyRole(currentUser, RoleThuKho) || isAdmin) {
			return &forecastTransitionError{status: http.StatusForbidden, message: "Only Thu kho (or Admin) can submit forecasts to Chi huy khoa"}
		}
		if normalizedExistingStatus != "" && normalizedExistingStatus != models.ForecastApprovalStatusEdited {
			return &forecastTransitionError{status: http.StatusBadRequest, message: "Only pending or edited forecasts can be submitted"}
		}
		return nil

	case models.ForecastApprovalStatusApproved:
		if !(userHasAnyRole(currentUser, RoleChiHuyKhoa) || isAdmin) {
			return &forecastTransitionError{status: http.StatusForbidden, message: "Only Chi huy khoa (or Admin) can approve submitted forecasts"}
		}
		if normalizedExistingStatus != models.ForecastApprovalStatusSubmitted {
			return &forecastTransitionError{status: http.StatusBadRequest, message: "Only submitted forecasts can be approved"}
		}
		return nil

	case models.ForecastApprovalStatusRejected:
		if userHasAnyRole(currentUser, RoleChiHuyKhoa) || isAdmin {
			if normalizedExistingStatus != models.ForecastApprovalStatusSubmitted {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Only submitted forecasts can be rejected by Chi huy khoa or Admin"}
			}
			return nil
		}

		if userHasAnyRole(currentUser, RoleThuKho) {
			if normalizedExistingStatus == models.ForecastApprovalStatusApproved {
				return &forecastTransitionError{status: http.StatusBadRequest, message: "Approved forecast cannot be rejected by Thu kho"}
			}
			return nil
		}

		return &forecastTransitionError{status: http.StatusForbidden, message: "Only Thu kho, Chi huy khoa (or Admin) can reject forecasts"}

	default:
		return &forecastTransitionError{status: http.StatusBadRequest, message: "status is invalid"}
	}
}

func lookupExistingForecastStatus(req SaveForecastApprovalRequest, statusByItemKey map[string]string) string {
	primaryKey := forecastApprovalStatusKey(req.MaQuanLy, req.MaVtytCu)
	if primaryKey != "" {
		if status, ok := statusByItemKey[primaryKey]; ok {
			return status
		}
	}

	fallbackKey := strings.TrimSpace(req.MaVtytCu)
	if fallbackKey == "" {
		return ""
	}

	return statusByItemKey[fallbackKey]
}

func forecastApprovalStatusKey(maQuanLy, maVtytCu string) string {
	normalizedMaQuanLy := strings.TrimSpace(maQuanLy)
	normalizedMaVtytCu := strings.TrimSpace(maVtytCu)

	if normalizedMaVtytCu != "" && normalizedMaQuanLy != "" {
		return normalizedMaVtytCu + "::" + normalizedMaQuanLy
	}

	if normalizedMaVtytCu != "" {
		return normalizedMaVtytCu
	}

	return normalizedMaQuanLy
}

// validateForecastMonthNotInPast checks if the given month/year is not in the past
// Returns an error if the month/year is before the current month/year
func validateForecastMonthNotInPast(month, year int) error {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	if year < currentYear || (year == currentYear && month < currentMonth) {
		return fmt.Errorf("cannot modify forecasts for past months (current month: %d/%d, requested: %d/%d)", currentMonth, currentYear, month, year)
	}

	return nil
}
