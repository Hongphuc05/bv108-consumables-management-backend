package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
)

type SupplyTaskHandler struct {
	supplyRepo *models.SupplyRepository
	taskRepo   *models.SupplyTaskRepository
	userRepo   *models.UserRepository
	jwtSecret  []byte
}

type updateSupplyVisibilityRequest struct {
	HideForOtherRoles bool `json:"hideForOtherRoles"`
}

type updateUserAssignmentsRequest struct {
	UserID         int64 `json:"userId"`
	SupplyIDX1List []int `json:"supplyIdx1List"`
}

func NewSupplyTaskHandler(
	supplyRepo *models.SupplyRepository,
	taskRepo *models.SupplyTaskRepository,
	userRepo *models.UserRepository,
	jwtSecret string,
) *SupplyTaskHandler {
	return &SupplyTaskHandler{
		supplyRepo: supplyRepo,
		taskRepo:   taskRepo,
		userRepo:   userRepo,
		jwtSecret:  []byte(jwtSecret),
	}
}

func (h *SupplyTaskHandler) authorizeManager(c *gin.Context) (*models.UserProfile, bool) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: "Yêu cầu đăng nhập hợp lệ"})
		return nil, false
	}

	if !userHasAnyRole(currentUser, RoleAdmin, RoleChiHuyKhoa) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "FORBIDDEN", Message: "Chỉ Admin hoặc Chỉ huy khoa mới có quyền truy cập tác vụ"})
		return nil, false
	}

	return currentUser, true
}

func (h *SupplyTaskHandler) GetState(c *gin.Context) {
	if _, ok := h.authorizeManager(c); !ok {
		return
	}

	hideEnabled, err := h.taskRepo.IsHideForOtherRolesEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	users, err := h.userRepo.ListOperationalUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	assignedCounts, err := h.taskRepo.GetAssignedCountsByUserIDs(userIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	totalSupplies, err := h.supplyRepo.CountAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	type userState struct {
		ID            int64  `json:"id"`
		Username      string `json:"username"`
		Email         string `json:"email"`
		Role          string `json:"role"`
		AssignedCount int    `json:"assignedCount"`
	}

	states := make([]userState, 0, len(users))
	for _, user := range users {
		states = append(states, userState{
			ID:            user.ID,
			Username:      user.Username,
			Email:         user.Email,
			Role:          user.Role,
			AssignedCount: assignedCounts[user.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"hideForOtherRoles": hideEnabled,
		"totalSupplies":     totalSupplies,
		"users":             states,
	})
}

func (h *SupplyTaskHandler) UpdateVisibility(c *gin.Context) {
	currentUser, ok := h.authorizeManager(c)
	if !ok {
		return
	}

	var req updateSupplyVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Payload cập nhật hiển thị không hợp lệ"})
		return
	}

	if err := h.taskRepo.SetHideForOtherRolesEnabled(req.HideForOtherRoles, currentUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật tùy chọn hiển thị vật tư thành công"})
}

func (h *SupplyTaskHandler) GetAssignmentsByUser(c *gin.Context) {
	if _, ok := h.authorizeManager(c); !ok {
		return
	}

	userID, err := strconv.ParseInt(strings.TrimSpace(c.Query("userId")), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: "userId không hợp lệ"})
		return
	}

	assignments, err := h.taskRepo.GetAssignedSupplyDetailsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":      userID,
		"assignments": assignments,
	})
}

func (h *SupplyTaskHandler) UpdateAssignmentsByUser(c *gin.Context) {
	currentUser, ok := h.authorizeManager(c)
	if !ok {
		return
	}

	var req updateUserAssignmentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Payload phân công không hợp lệ"})
		return
	}

	if req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: "userId không hợp lệ"})
		return
	}

	uniqueSupplyMap := make(map[int]struct{})
	uniqueSupplyIDs := make([]int, 0, len(req.SupplyIDX1List))
	for _, idx1 := range req.SupplyIDX1List {
		if idx1 <= 0 {
			continue
		}
		if _, exists := uniqueSupplyMap[idx1]; exists {
			continue
		}
		uniqueSupplyMap[idx1] = struct{}{}
		uniqueSupplyIDs = append(uniqueSupplyIDs, idx1)
	}
	sort.Ints(uniqueSupplyIDs)

	if err := h.taskRepo.ReplaceAssignmentsForUser(req.UserID, uniqueSupplyIDs, currentUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lưu phân công vật tư thành công"})
}

func (h *SupplyTaskHandler) GetSupplyCatalog(c *gin.Context) {
	if _, ok := h.authorizeManager(c); !ok {
		return
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	const assignmentCatalogMax = 5000

	var (
		supplies []models.Supply
		total    int
		err      error
	)

	if keyword == "" {
		supplies, total, err = h.supplyRepo.GetAllVisible(1, assignmentCatalogMax, nil)
	} else {
		supplies, total, err = h.supplyRepo.SearchByNameVisible(keyword, 1, assignmentCatalogMax, nil)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  supplies,
		"total": total,
	})
}
