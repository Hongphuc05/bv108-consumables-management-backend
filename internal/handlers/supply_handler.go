package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"bv108-consumables-management-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type SupplyHandler struct {
	repo      *models.SupplyRepository
	userRepo  *models.UserRepository
	taskRepo  *models.SupplyTaskRepository
	jwtSecret []byte
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 10000
)

var compareSupplyExcelHeaders = []string{
	"STT",
	"Tên công ty",
	"Mã thư viện",
	"Mã Thông tư 04",
	"Tên vật tư",
	"Tên thương mại",
	"TSKT 2025",
	"TSKT 2026",
	"Chất liệu/ Vật liệu",
	"Đặc tính/Cấu tạo",
	"Kích thước",
	"Chiều dài",
	"Tính năng sử dụng",
	"TSKT khác",
	"ĐVT",
	"Số lượng sử dụng 12 tháng",
	"Số lượng trúng thầu 2025 + bổ sung",
	"Đơn giá trúng thầu năm 2025",
	"Đơn giá đề xuất năm 2026",
	"KQ trúng thầu THẤP NHẤT",
	"TG/ĐV đăng tải giá THẤP NHẤT",
	"KQ trúng thầu CAO NHẤT",
	"TG/ĐV đăng tải giá CAO NHẤT",
	"Mã số thuế",
	"Mã hiệu",
	"Hãng sản xuất",
	"Nước sản xuất",
	"Nhóm nước",
	"Chất lượng",
	"Mã 5086",
}

// NewSupplyHandler creates a new supply handler
func NewSupplyHandler(
	repo *models.SupplyRepository,
	userRepo *models.UserRepository,
	taskRepo *models.SupplyTaskRepository,
	jwtSecret string,
) *SupplyHandler {
	return &SupplyHandler{
		repo:      repo,
		userRepo:  userRepo,
		taskRepo:  taskRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h *SupplyHandler) getVisibleSupplyIDX1ForRequester(c *gin.Context) ([]int, bool) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "Yêu cầu đăng nhập hợp lệ",
		})
		return nil, false
	}

	if userHasAnyRole(currentUser, RoleAdmin, RoleChiHuyKhoa) {
		return nil, true
	}

	hideForOtherRoles, err := h.taskRepo.IsHideForOtherRolesEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return nil, false
	}

	if !hideForOtherRoles {
		return nil, true
	}

	if !shouldRestrictSupplyVisibilityByAssignment(currentUser.Role) {
		return nil, true
	}

	visibleIDX1, err := h.taskRepo.GetAssignedSupplyIDX1ByUserID(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return nil, false
	}

	return visibleIDX1, true
}

func shouldRestrictSupplyVisibilityByAssignment(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleNhanVienThau:
		return true
	default:
		return false
	}
}

func (h *SupplyHandler) requireAuthenticatedRequester(c *gin.Context) bool {
	if _, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "Yêu cầu đăng nhập hợp lệ",
		})
		return false
	}

	return true
}

func (h *SupplyHandler) getAuthenticatedRequester(c *gin.Context) (*models.UserProfile, bool) {
	currentUser, err := getCurrentUserFromAuthorizationHeader(c, h.userRepo, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "UNAUTHORIZED",
			Message: "Yêu cầu đăng nhập hợp lệ",
		})
		return nil, false
	}

	return currentUser, true
}

func nullableStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableFloatValue(value sql.NullFloat64) float64 {
	if !value.Valid {
		return 0
	}
	return value.Float64
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

type CompareRequest struct {
	MaThuVien []string `json:"maThuVien"`
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(defaultPage)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))

	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)

	supplies, total, err := h.repo.GetAllVisible(page, pageSize, visibleIDX1)
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_ID",
			Message: "Invalid supply ID",
		})
		return
	}

	supply, err := h.repo.GetByIDVisible(id, visibleIDX1)
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MISSING_KEYWORD",
			Message: "Search keyword is required",
		})
		return
	}

	page, pageSize := parsePagination(c)

	supplies, total, err := h.repo.SearchByNameVisible(keyword, page, pageSize, visibleIDX1)
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	groupName := c.Query("groupName")
	if groupName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "MISSING_GROUP",
			Message: "Group name is required",
		})
		return
	}

	page, pageSize := parsePagination(c)

	supplies, total, err := h.repo.GetByGroupVisible(groupName, page, pageSize, visibleIDX1)
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	groups, err := h.repo.GetAllGroupsVisible(visibleIDX1)
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
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	threshold, _ := strconv.Atoi(c.DefaultQuery("threshold", "20"))
	page, pageSize := parsePagination(c)

	supplies, total, err := h.repo.GetLowStockVisible(threshold, page, pageSize, visibleIDX1)
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

func (h *SupplyHandler) ExportCompareCatalogExcel(c *gin.Context) {
	if !h.requireAuthenticatedRequester(c) {
		return
	}

	items, err := h.repo.ListAllCompareSupplies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	workbook := excelize.NewFile()
	sheetName := "so_sanh_vat_tu"
	workbook.SetSheetName(workbook.GetSheetName(0), sheetName)

	for colIndex, header := range compareSupplyExcelHeaders {
		cell, cellErr := excelize.CoordinatesToCellName(colIndex+1, 1)
		if cellErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "EXPORT_ERROR",
				Message: cellErr.Error(),
			})
			return
		}
		if setErr := workbook.SetCellValue(sheetName, cell, header); setErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "EXPORT_ERROR",
				Message: setErr.Error(),
			})
			return
		}
	}

	for rowIndex, item := range items {
		row := []interface{}{
			item.STT,
			nullableStringValue(item.TenCongTy),
			nullableStringValue(item.MaThuVien),
			nullableStringValue(item.MaThongTu04),
			nullableStringValue(item.TenVatTu),
			nullableStringValue(item.TenThuongMai),
			nullableStringValue(item.TSKT2025),
			nullableStringValue(item.TSKT2026),
			nullableStringValue(item.ChatLieuVatLieu),
			nullableStringValue(item.DacTinhCauTao),
			nullableStringValue(item.KichThuoc),
			nullableStringValue(item.ChieuDai),
			nullableStringValue(item.TinhNangSuDung),
			nullableStringValue(item.TSKTKhac),
			nullableStringValue(item.DVT),
			nullableFloatValue(item.SoLuongSuDung12Thang),
			nullableFloatValue(item.SoLuongTrungThau2025BoSung),
			nullableFloatValue(item.DonGiaTrungThau2025),
			nullableFloatValue(item.DonGiaDeXuat2026),
			nullableFloatValue(item.KetQuaTrungThauThapNhat),
			nullableStringValue(item.ThoiGianDangTaiThapNhat),
			nullableFloatValue(item.KetQuaTrungThauCaoNhat),
			nullableStringValue(item.ThoiGianDangTaiCaoNhat),
			nullableStringValue(item.MaSoThue),
			nullableStringValue(item.MaHieu),
			nullableStringValue(item.HangSX),
			nullableStringValue(item.NuocSX),
			nullableStringValue(item.NhomNuoc),
			nullableStringValue(item.ChatLuong),
			nullableStringValue(item.Ma5086),
		}

		startCell, cellErr := excelize.CoordinatesToCellName(1, rowIndex+2)
		if cellErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "EXPORT_ERROR",
				Message: cellErr.Error(),
			})
			return
		}
		if setErr := workbook.SetSheetRow(sheetName, startCell, &row); setErr != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "EXPORT_ERROR",
				Message: setErr.Error(),
			})
			return
		}
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="so-sanh-vat-tu-template.xlsx"`)
	c.Header("Cache-Control", "no-store")

	if err := workbook.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "EXPORT_ERROR",
			Message: err.Error(),
		})
		return
	}
}

func (h *SupplyHandler) ImportCompareCatalogExcel(c *gin.Context) {
	currentUser, ok := h.getAuthenticatedRequester(c)
	if !ok {
		return
	}

	if !userHasAnyRole(currentUser, RoleAdmin, RoleChiHuyKhoa) {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "FORBIDDEN",
			Message: "Chỉ Admin hoặc Chỉ huy khoa mới có quyền import thay thế dữ liệu so sánh",
		})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_FILE",
			Message: "Thiếu file Excel import",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_FILE",
			Message: "Không mở được file Excel import",
		})
		return
	}
	defer file.Close()

	workbook, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_FILE",
			Message: "File import phải là Excel .xlsx hợp lệ",
		})
		return
	}
	defer workbook.Close()

	sheetName := workbook.GetSheetName(0)
	if strings.TrimSpace(sheetName) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_FILE",
			Message: "File Excel không có sheet dữ liệu",
		})
		return
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_FILE",
			Message: "Không đọc được dữ liệu từ file Excel",
		})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "EMPTY_FILE",
			Message: "File Excel không có dữ liệu",
		})
		return
	}

	headerRow := make([]string, len(compareSupplyExcelHeaders))
	for i := range compareSupplyExcelHeaders {
		if i < len(rows[0]) {
			headerRow[i] = strings.TrimSpace(rows[0][i])
		}
	}

	for i, expected := range compareSupplyExcelHeaders {
		if headerRow[i] != expected {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_TEMPLATE",
				Message: fmt.Sprintf("Header cột %d phải là %q", i+1, expected),
			})
			return
		}
	}

	inputs := make([]models.CompareSupplyReplaceInput, 0, len(rows)-1)
	seenLibraryCodes := make(map[string]bool)

	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		cells := make([]string, len(compareSupplyExcelHeaders))
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

		stt, parseErr := strconv.Atoi(cells[0])
		if parseErr != nil || stt <= 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_DATA",
				Message: fmt.Sprintf("Dòng %d có STT không hợp lệ", rowIndex+1),
			})
			return
		}

		maThuVien := cells[2]
		if maThuVien == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "INVALID_DATA",
				Message: fmt.Sprintf("Dòng %d thiếu Mã thư viện", rowIndex+1),
			})
			return
		}
		if seenLibraryCodes[maThuVien] {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "DUPLICATE_CODE",
				Message: fmt.Sprintf("Mã thư viện %q bị lặp trong file import", maThuVien),
			})
			return
		}
		seenLibraryCodes[maThuVien] = true

		parseFloat := func(raw string, fieldName string) (float64, error) {
			normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
			if normalized == "" {
				return 0, nil
			}
			value, err := strconv.ParseFloat(normalized, 64)
			if err != nil {
				return 0, fmt.Errorf("%s không hợp lệ", fieldName)
			}
			return value, nil
		}

		soLuongSuDung12Thang, err := parseFloat(cells[15], "Số lượng sử dụng 12 tháng")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}
		soLuongTrungThau2025BoSung, err := parseFloat(cells[16], "Số lượng trúng thầu 2025 + bổ sung")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}
		donGiaTrungThau2025, err := parseFloat(cells[17], "Đơn giá trúng thầu năm 2025")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}
		donGiaDeXuat2026, err := parseFloat(cells[18], "Đơn giá đề xuất năm 2026")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}
		ketQuaTrungThauThapNhat, err := parseFloat(cells[19], "KQ trúng thầu THẤP NHẤT")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}
		ketQuaTrungThauCaoNhat, err := parseFloat(cells[21], "KQ trúng thầu CAO NHẤT")
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_DATA", Message: fmt.Sprintf("Dòng %d: %s", rowIndex+1, err.Error())})
			return
		}

		inputs = append(inputs, models.CompareSupplyReplaceInput{
			STT:                        stt,
			TenCongTy:                  cells[1],
			MaThuVien:                  maThuVien,
			MaThongTu04:                cells[3],
			TenVatTu:                   cells[4],
			TenThuongMai:               cells[5],
			TSKT2025:                   cells[6],
			TSKT2026:                   cells[7],
			ChatLieuVatLieu:            cells[8],
			DacTinhCauTao:              cells[9],
			KichThuoc:                  cells[10],
			ChieuDai:                   cells[11],
			TinhNangSuDung:             cells[12],
			TSKTKhac:                   cells[13],
			DVT:                        cells[14],
			SoLuongSuDung12Thang:       soLuongSuDung12Thang,
			SoLuongTrungThau2025BoSung: soLuongTrungThau2025BoSung,
			DonGiaTrungThau2025:        donGiaTrungThau2025,
			DonGiaDeXuat2026:           donGiaDeXuat2026,
			KetQuaTrungThauThapNhat:    ketQuaTrungThauThapNhat,
			ThoiGianDangTaiThapNhat:    cells[20],
			KetQuaTrungThauCaoNhat:     ketQuaTrungThauCaoNhat,
			ThoiGianDangTaiCaoNhat:     cells[22],
			MaSoThue:                   cells[23],
			MaHieu:                     cells[24],
			HangSX:                     cells[25],
			NuocSX:                     cells[26],
			NhomNuoc:                   cells[27],
			ChatLuong:                  cells[28],
			Ma5086:                     cells[29],
		})
	}

	if len(inputs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "EMPTY_FILE",
			Message: "File Excel không có dòng dữ liệu hợp lệ để import",
		})
		return
	}

	if err := h.repo.ReplaceAllCompareSupplies(inputs); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Import dữ liệu so sánh thành công",
		"count":   len(inputs),
	})
}

// GetCompareCatalog returns paginated rows from so_sanh_vat_tu for selection list.
func (h *SupplyHandler) GetCompareCatalog(c *gin.Context) {
	if !h.requireAuthenticatedRequester(c) {
		return
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	level1Filter := strings.TrimSpace(c.Query("level1Filter"))
	level2Filter := strings.TrimSpace(c.Query("level2Filter"))
	page, pageSize := parsePagination(c)

	items, total, err := h.repo.GetCompareCatalog(keyword, level1Filter, level2Filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	c.JSON(http.StatusOK, PaginationResponse{
		Data:       items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetCompareLevel1Options returns distinct level 1 values from ma_thong_tu_04.
func (h *SupplyHandler) GetCompareLevel1Options(c *gin.Context) {
	if !h.requireAuthenticatedRequester(c) {
		return
	}

	groups, err := h.repo.GetCompareLevel1Options()
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

// GetCompareLevel2Options returns level 2 values (last 3 chars) from ma_thong_tu_04.
func (h *SupplyHandler) GetCompareLevel2Options(c *gin.Context) {
	if !h.requireAuthenticatedRequester(c) {
		return
	}

	level1 := strings.TrimSpace(c.Query("level1"))
	groups, err := h.repo.GetCompareLevel2Options(level1)
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

// CompareSupplies returns selected comparison rows by ma_thu_vien.
func (h *SupplyHandler) CompareSupplies(c *gin.Context) {
	if !h.requireAuthenticatedRequester(c) {
		return
	}

	var req CompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid compare payload",
		})
		return
	}

	if len(req.MaThuVien) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_SELECTION",
			Message: "Vui long chon it nhat 2 vat tu de so sanh",
		})
		return
	}

	if len(req.MaThuVien) > 10 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_SELECTION",
			Message: "Chi duoc so sanh toi da 10 vat tu moi lan",
		})
		return
	}

	unique := make([]string, 0, len(req.MaThuVien))
	seen := make(map[string]bool)
	for _, raw := range req.MaThuVien {
		code := strings.TrimSpace(raw)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		unique = append(unique, code)
	}

	if len(unique) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_SELECTION",
			Message: "Vui long chon it nhat 2 ma thu vien hop le",
		})
		return
	}

	items, err := h.repo.GetCompareByLibraryCodes(unique)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": len(items),
	})
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Success 200 {object} map[string]string
// @Router /health [get]
// GetForecastCatalog retrieves non-zero supplies for forecast processing
// @Summary Get forecast supply catalog
// @Description Get supplies that have inventory activity, optimized for the forecast screen
// @Tags supplies
// @Param keyword query string false "Search keyword"
// @Success 200 {object} gin.H
// @Failure 500 {object} ErrorResponse
// @Router /api/supplies/forecast-catalog [get]
func (h *SupplyHandler) GetForecastCatalog(c *gin.Context) {
	visibleIDX1, ok := h.getVisibleSupplyIDX1ForRequester(c)
	if !ok {
		return
	}

	keyword := c.Query("keyword")

	supplies, err := h.repo.GetForecastCatalogVisible(keyword, visibleIDX1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  supplies,
		"total": len(supplies),
	})
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "OK",
		"message": "BV108 Consumables API is running",
	})
}
