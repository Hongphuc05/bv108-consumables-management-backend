package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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

type importSupplyAssignmentsItemRequest struct {
	IDX1   int    `json:"idx1"`
	UserID *int64 `json:"userId"`
}

type importSupplyAssignmentsRequest struct {
	Items []importSupplyAssignmentsItemRequest `json:"items"`
}

var supplyTaskAssignmentExcelHeaders = []string{
	"IDX1",
	"PRODUCTID",
	"GROUPNAME",
	"ID",
	"IDX2",
	"MA_HIEU",
	"TYPENAME",
	"NAME",
	"UNIT",
	"QUY_CACH_DONG_GOI",
	"QUY_CACH_GIAO_HANG",
	"QUY_CACH_TOI_THIEU",
	"THONG_TIN_THAU",
	"TONGTHAU",
	"HANGSX",
	"NUOC_SX",
	"NHA_CUNG_CAP",
	"PRICE",
	"TONDAUKY",
	"NHAPTRONGKY",
	"XUATTRONGKY",
	"TONGNHAP",
	"TON_KHO_MIN",
	"id_thu_ki_phu_trach",
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

	targetUser, err := loadSupplyAssignmentUser(h.userRepo, userID)
	if err != nil {
		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "Người dùng không tồn tại"})
			return
		case "user account is disabled":
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "ACCOUNT_DISABLED", Message: "Tài khoản đã bị vô hiệu hóa"})
			return
		case "user is not eligible for supply assignments":
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: "Chỉ Nhân viên thầu mới được nhận phân công vật tư"})
			return
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
			return
		}
	}

	assignments, err := h.taskRepo.GetAssignedSupplyDetailsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":      userID,
		"username":    targetUser.Username,
		"userRole":    targetUser.Role,
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

	if _, err := loadSupplyAssignmentUser(h.userRepo, req.UserID); err != nil {
		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "Người dùng không tồn tại"})
			return
		case "user account is disabled":
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "ACCOUNT_DISABLED", Message: "Tài khoản đã bị vô hiệu hóa"})
			return
		case "user is not eligible for supply assignments":
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: "Chỉ Nhân viên thầu mới được nhận phân công vật tư"})
			return
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
			return
		}
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

func (h *SupplyTaskHandler) ExportAssignments(c *gin.Context) {
	if _, ok := h.authorizeManager(c); !ok {
		return
	}

	rows, err := h.taskRepo.ListExportRows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	filename := fmt.Sprintf("phan-quyen-vat-tu-%s.xlsx", time.Now().Format("20060102-150405"))
	workbook := excelize.NewFile()
	sheetName := "phan_quyen_vat_tu"
	workbook.SetSheetName(workbook.GetSheetName(0), sheetName)

	for colIndex, header := range supplyTaskAssignmentExcelHeaders {
		cell, cellErr := excelize.CoordinatesToCellName(colIndex+1, 1)
		if cellErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WRITE_ERROR", Message: "Không thể tạo ô tiêu đề export"})
			return
		}
		if err := workbook.SetCellValue(sheetName, cell, header); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WRITE_ERROR", Message: "Không thể ghi tiêu đề file export"})
			return
		}
	}

	for rowIndex, row := range rows {
		record := []interface{}{
			row.IDX1,
			nullInt32Value(row.ProductID),
			formatNullString(row.GroupName),
			formatNullString(row.ID),
			formatNullString(row.IDX2),
			formatNullString(row.MaHieu),
			formatNullString(row.TypeName),
			formatNullString(row.Name),
			formatNullString(row.Unit),
			formatNullString(row.QuyCachDongGoi),
			formatNullString(row.QuyCachGiaoHang),
			formatNullString(row.QuyCachToiThieu),
			formatNullString(row.ThongTinThau),
			formatNullString(row.TongThau),
			formatNullString(row.HangSX),
			formatNullString(row.NuocSX),
			formatNullString(row.NhaCungCap),
			nullFloat64Value(row.Price),
			nullInt32Value(row.TonDauKy),
			nullInt32Value(row.NhapTrongKy),
			nullInt32Value(row.XuatTrongKy),
			nullInt32Value(row.TongNhap),
			nullInt32Value(row.TonKhoMin),
			nullInt64Value(row.AssignedToUserID),
		}

		startCell, cellErr := excelize.CoordinatesToCellName(1, rowIndex+2)
		if cellErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WRITE_ERROR", Message: "Không thể tạo ô dữ liệu export"})
			return
		}
		if err := workbook.SetSheetRow(sheetName, startCell, &record); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WRITE_ERROR", Message: "Không thể ghi dữ liệu export"})
			return
		}
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-store")

	if err := workbook.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WRITE_ERROR", Message: "Không thể hoàn tất file export"})
		return
	}
}

func (h *SupplyTaskHandler) ImportAssignments(c *gin.Context) {
	currentUser, ok := h.authorizeManager(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_FILE", Message: "Thiếu file Excel import phân công"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_FILE", Message: "Không mở được file Excel import phân công"})
		return
	}
	defer file.Close()

	workbook, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_FILE", Message: "File import phải là Excel .xlsx hợp lệ"})
		return
	}
	defer workbook.Close()

	sheetName := workbook.GetSheetName(0)
	if strings.TrimSpace(sheetName) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_FILE", Message: "File Excel không có sheet dữ liệu"})
		return
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_FILE", Message: "Không đọc được dữ liệu từ file Excel"})
		return
	}

	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "EMPTY_IMPORT", Message: "File import không có dòng dữ liệu hợp lệ"})
		return
	}

	headerRow := make([]string, len(supplyTaskAssignmentExcelHeaders))
	for i := range supplyTaskAssignmentExcelHeaders {
		if i < len(rows[0]) {
			headerRow[i] = strings.TrimSpace(rows[0][i])
		}
	}

	for i, expected := range supplyTaskAssignmentExcelHeaders {
		if headerRow[i] != expected {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_TEMPLATE", Message: fmt.Sprintf("Header cột %d phải là %q", i+1, expected)})
			return
		}
	}

	req := importSupplyAssignmentsRequest{
		Items: make([]importSupplyAssignmentsItemRequest, 0, len(rows)-1),
	}

	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		cells := make([]string, len(supplyTaskAssignmentExcelHeaders))
		for i := range cells {
			if i < len(row) {
				cells[i] = strings.TrimSpace(row[i])
			}
		}

		isEmpty := true
		for _, cell := range cells {
			if cell != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		idx1, err := strconv.Atoi(cells[0])
		if err != nil || idx1 <= 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_SUPPLY", Message: fmt.Sprintf("Dòng %d có IDX1 không hợp lệ", rowIndex+1)})
			return
		}

		nextItem := importSupplyAssignmentsItemRequest{IDX1: idx1}
		if strings.TrimSpace(cells[23]) != "" {
			userID, err := strconv.ParseInt(strings.TrimSpace(cells[23]), 10, 64)
			if err != nil || userID <= 0 {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: fmt.Sprintf("Dòng %d có id_thu_ki_phu_trach không hợp lệ", rowIndex+1)})
				return
			}
			nextItem.UserID = &userID
		}

		req.Items = append(req.Items, nextItem)
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "EMPTY_IMPORT", Message: "File import không có dòng dữ liệu hợp lệ"})
		return
	}

	activeUsers, err := h.userRepo.ListActiveUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	activeUserMap := make(map[int64]models.UserProfile, len(activeUsers))
	for _, user := range activeUsers {
		activeUserMap[user.ID] = user
	}

	validUsers, err := h.userRepo.ListOperationalUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	validUserMap := make(map[int64]struct{}, len(validUsers))
	for _, user := range validUsers {
		validUserMap[user.ID] = struct{}{}
	}

	seenIDX1 := make(map[int]struct{}, len(req.Items))
	supplyIDX1List := make([]int, 0, len(req.Items))
	assignments := make([]models.SupplyTaskImportAssignment, 0, len(req.Items))

	for _, item := range req.Items {
		if item.IDX1 <= 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_SUPPLY", Message: "Có dòng import chứa IDX1 không hợp lệ"})
			return
		}

		if _, exists := seenIDX1[item.IDX1]; exists {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "DUPLICATE_SUPPLY", Message: fmt.Sprintf("IDX1 %d bị lặp trong file import", item.IDX1)})
			return
		}
		seenIDX1[item.IDX1] = struct{}{}
		supplyIDX1List = append(supplyIDX1List, item.IDX1)

		nextAssignment := models.SupplyTaskImportAssignment{SupplyIDX1: item.IDX1}
		if item.UserID != nil {
			if *item.UserID <= 0 {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: fmt.Sprintf("IDX1 %d có userId không hợp lệ", item.IDX1)})
				return
			}
			activeUser, exists := activeUserMap[*item.UserID]
			if !exists {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: fmt.Sprintf("IDX1 %d tham chiếu userId %d không tồn tại hoặc đã ngừng hoạt động", item.IDX1, *item.UserID)})
				return
			}
			if _, exists := validUserMap[*item.UserID]; !exists {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_USER", Message: fmt.Sprintf("IDX1 %d tham chiếu userId %d là %s nên không được phép nhận phân công vật tư", item.IDX1, *item.UserID, formatRoleLabelForPermissions(activeUser.Role))})
				return
			}
			nextAssignment.UserID = *item.UserID
			nextAssignment.Assigned = true
		}

		assignments = append(assignments, nextAssignment)
	}

	existingIDX1Map, err := h.taskRepo.GetExistingSupplyIDX1Set(supplyIDX1List)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	for _, idx1 := range supplyIDX1List {
		if _, exists := existingIDX1Map[idx1]; !exists {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_SUPPLY", Message: fmt.Sprintf("IDX1 %d không tồn tại trong danh mục vật tư", idx1)})
			return
		}
	}

	assignedCount, clearedCount, err := h.taskRepo.ReplaceAssignmentsBySupplyIDX1(assignments, currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Import phân công vật tư thành công",
		"updatedCount":  len(assignments),
		"assignedCount": assignedCount,
		"clearedCount":  clearedCount,
	})
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

func formatNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullInt32Value(value sql.NullInt32) interface{} {
	if !value.Valid {
		return ""
	}
	return value.Int32
}

func nullInt64Value(value sql.NullInt64) interface{} {
	if !value.Valid {
		return ""
	}
	return value.Int64
}

func nullFloat64Value(value sql.NullFloat64) interface{} {
	if !value.Valid {
		return ""
	}
	return value.Float64
}
