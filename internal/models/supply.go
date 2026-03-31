package models

import (
	"database/sql"
	"fmt"
	"strings"
)

// Supply represents a medical supply item from the database
type Supply struct {
	IDX1         int             `json:"idx1"`
	ProductID    sql.NullInt32   `json:"productId"`
	GroupName    sql.NullString  `json:"groupName"`
	ID           sql.NullString  `json:"id"`
	IDX2         sql.NullString  `json:"idx2"`
	MaHieu       sql.NullString  `json:"maHieu"`
	TypeName     sql.NullString  `json:"typeName"`
	Name         sql.NullString  `json:"name"`
	Unit         sql.NullString  `json:"unit"`
	QuyCach      sql.NullString  `json:"quyCach"`
	ThongTinThau sql.NullString  `json:"thongTinThau"`
	TongThau     sql.NullString  `json:"tongThau"`
	HangSX       sql.NullString  `json:"hangSx"`
	NuocSX       sql.NullString  `json:"nuocSx"`
	NhaCungCap   sql.NullString  `json:"nhaCungCap"`
	Price        sql.NullFloat64 `json:"price"`
	TonDauKy     sql.NullInt32   `json:"tonDauKy"`
	NhapTrongKy  sql.NullInt32   `json:"nhapTrongKy"`
	XuatTrongKy  sql.NullInt32   `json:"xuatTrongKy"`
	TongNhap     sql.NullInt32   `json:"tongNhap"`
	// Calculated field
	TonCuoiKy int `json:"tonCuoiKy"`
}

// CompareSupply represents one row in so_sanh_vat_tu table.
type CompareSupply struct {
	STT                        int             `json:"stt"`
	TenCongTy                  sql.NullString  `json:"tenCongTy"`
	MaThuVien                  sql.NullString  `json:"maThuVien"`
	MaThongTu04                sql.NullString  `json:"maThongTu04"`
	TenVatTu                   sql.NullString  `json:"tenVatTu"`
	TenThuongMai               sql.NullString  `json:"tenThuongMai"`
	TSKT2025                   sql.NullString  `json:"tskt2025"`
	TSKT2026                   sql.NullString  `json:"tskt2026"`
	ChatLieuVatLieu            sql.NullString  `json:"chatLieuVatLieu"`
	DacTinhCauTao              sql.NullString  `json:"dacTinhCauTao"`
	KichThuoc                  sql.NullString  `json:"kichThuoc"`
	ChieuDai                   sql.NullString  `json:"chieuDai"`
	TinhNangSuDung             sql.NullString  `json:"tinhNangSuDung"`
	TSKTKhac                   sql.NullString  `json:"tsktKhac"`
	DVT                        sql.NullString  `json:"dvt"`
	SoLuongSuDung12Thang       sql.NullFloat64 `json:"soLuongSuDung12Thang"`
	SoLuongTrungThau2025BoSung sql.NullFloat64 `json:"soLuongTrungThau2025BoSung"`
	DonGiaTrungThau2025        sql.NullFloat64 `json:"donGiaTrungThau2025"`
	DonGiaDeXuat2026           sql.NullFloat64 `json:"donGiaDeXuat2026"`
	KetQuaTrungThauThapNhat    sql.NullFloat64 `json:"ketQuaTrungThauThapNhat"`
	ThoiGianDangTaiThapNhat    sql.NullString  `json:"thoiGianDangTaiThapNhat"`
	KetQuaTrungThauCaoNhat     sql.NullFloat64 `json:"ketQuaTrungThauCaoNhat"`
	ThoiGianDangTaiCaoNhat     sql.NullString  `json:"thoiGianDangTaiCaoNhat"`
	MaSoThue                   sql.NullString  `json:"maSoThue"`
	MaHieu                     sql.NullString  `json:"maHieu"`
	HangSX                     sql.NullString  `json:"hangSx"`
	NuocSX                     sql.NullString  `json:"nuocSx"`
	NhomNuoc                   sql.NullString  `json:"nhomNuoc"`
	ChatLuong                  sql.NullString  `json:"chatLuong"`
	Ma5086                     sql.NullString  `json:"ma5086"`
	CreatedAt                  sql.NullTime    `json:"createdAt"`
	UpdatedAt                  sql.NullTime    `json:"updatedAt"`
}

func scanCompareSupplyRow(scanner interface {
	Scan(dest ...interface{}) error
}) (CompareSupply, error) {
	var item CompareSupply
	err := scanner.Scan(
		&item.STT,
		&item.TenCongTy,
		&item.MaThuVien,
		&item.MaThongTu04,
		&item.TenVatTu,
		&item.TenThuongMai,
		&item.TSKT2025,
		&item.TSKT2026,
		&item.ChatLieuVatLieu,
		&item.DacTinhCauTao,
		&item.KichThuoc,
		&item.ChieuDai,
		&item.TinhNangSuDung,
		&item.TSKTKhac,
		&item.DVT,
		&item.SoLuongSuDung12Thang,
		&item.SoLuongTrungThau2025BoSung,
		&item.DonGiaTrungThau2025,
		&item.DonGiaDeXuat2026,
		&item.KetQuaTrungThauThapNhat,
		&item.ThoiGianDangTaiThapNhat,
		&item.KetQuaTrungThauCaoNhat,
		&item.ThoiGianDangTaiCaoNhat,
		&item.MaSoThue,
		&item.MaHieu,
		&item.HangSX,
		&item.NuocSX,
		&item.NhomNuoc,
		&item.ChatLuong,
		&item.Ma5086,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	return item, err
}

// calculateTonCuoiKy calculates TonCuoiKy from nullable integer fields
func calculateTonCuoiKy(tonDauKy, nhapTrongKy, xuatTrongKy sql.NullInt32) int {
	tdk := 0
	ntk := 0
	xtk := 0

	if tonDauKy.Valid {
		tdk = int(tonDauKy.Int32)
	}
	if nhapTrongKy.Valid {
		ntk = int(nhapTrongKy.Int32)
	}
	if xuatTrongKy.Valid {
		xtk = int(xuatTrongKy.Int32)
	}

	return tdk + ntk - xtk
}

// SupplyRepository handles database operations for supplies
type SupplyRepository struct {
	DB *sql.DB
}

// NewSupplyRepository creates a new supply repository
func NewSupplyRepository(db *sql.DB) *SupplyRepository {
	return &SupplyRepository{DB: db}
}

// GetAll retrieves all supplies with pagination
func (r *SupplyRepository) GetAll(page, pageSize int) ([]Supply, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM supplies"
	err := r.DB.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	// Get paginated data
	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT, QUY_CACH_DONG_GOI AS QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		var s Supply
		err := rows.Scan(
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2, &s.MaHieu,
			&s.TypeName, &s.Name, &s.Unit, &s.QuyCach, &s.ThongTinThau, &s.TongThau,
			&s.HangSX, &s.NuocSX, &s.NhaCungCap, &s.Price,
			&s.TonDauKy, &s.NhapTrongKy, &s.XuatTrongKy, &s.TongNhap,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}

		s.TonCuoiKy = calculateTonCuoiKy(s.TonDauKy, s.NhapTrongKy, s.XuatTrongKy)
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

// GetByID retrieves a supply by IDX1
func (r *SupplyRepository) GetByID(idx1 int) (*Supply, error) {
	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT, QUY_CACH_DONG_GOI AS QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE IDX1 = ?
	`

	var s Supply
	err := r.DB.QueryRow(query, idx1).Scan(
		&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2, &s.MaHieu,
		&s.TypeName, &s.Name, &s.Unit, &s.QuyCach, &s.ThongTinThau, &s.TongThau,
		&s.HangSX, &s.NuocSX, &s.NhaCungCap, &s.Price,
		&s.TonDauKy, &s.NhapTrongKy, &s.XuatTrongKy, &s.TongNhap,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supply not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying supply: %w", err)
	}

	s.TonCuoiKy = calculateTonCuoiKy(s.TonDauKy, s.NhapTrongKy, s.XuatTrongKy)

	return &s, nil
}

// SearchByName searches supplies by name
func (r *SupplyRepository) SearchByName(keyword string, page, pageSize int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize
	searchPattern := "%" + keyword + "%"

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM supplies WHERE NAME LIKE ? OR ID LIKE ? OR IDX2 LIKE ? OR MA_HIEU LIKE ?"
	err := r.DB.QueryRow(countQuery, searchPattern, searchPattern, searchPattern, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	// Get paginated data
	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT, QUY_CACH_DONG_GOI AS QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE NAME LIKE ? OR ID LIKE ? OR IDX2 LIKE ? OR MA_HIEU LIKE ?
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, searchPattern, searchPattern, searchPattern, searchPattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		var s Supply
		err := rows.Scan(
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2, &s.MaHieu,
			&s.TypeName, &s.Name, &s.Unit, &s.QuyCach, &s.ThongTinThau, &s.TongThau,
			&s.HangSX, &s.NuocSX, &s.NhaCungCap, &s.Price,
			&s.TonDauKy, &s.NhapTrongKy, &s.XuatTrongKy, &s.TongNhap,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}

		s.TonCuoiKy = calculateTonCuoiKy(s.TonDauKy, s.NhapTrongKy, s.XuatTrongKy)
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

// GetByGroup retrieves supplies by group name
func (r *SupplyRepository) GetByGroup(groupName string, page, pageSize int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM supplies WHERE GROUPNAME = ?"
	err := r.DB.QueryRow(countQuery, groupName).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	// Get paginated data
	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT, QUY_CACH_DONG_GOI AS QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE GROUPNAME = ?
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, groupName, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		var s Supply
		err := rows.Scan(
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2, &s.MaHieu,
			&s.TypeName, &s.Name, &s.Unit, &s.QuyCach, &s.ThongTinThau, &s.TongThau,
			&s.HangSX, &s.NuocSX, &s.NhaCungCap, &s.Price,
			&s.TonDauKy, &s.NhapTrongKy, &s.XuatTrongKy, &s.TongNhap,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}

		s.TonCuoiKy = calculateTonCuoiKy(s.TonDauKy, s.NhapTrongKy, s.XuatTrongKy)
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

// GetAllGroups retrieves all unique group names
func (r *SupplyRepository) GetAllGroups() ([]string, error) {
	query := "SELECT DISTINCT GROUPNAME FROM supplies ORDER BY GROUPNAME"

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying groups: %w", err)
	}
	defer rows.Close()

	groups := []string{}
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("error scanning group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetLowStock retrieves supplies with low stock (TonCuoiKy < threshold)
func (r *SupplyRepository) GetLowStock(threshold int, page, pageSize int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize

	// For low stock, we need to calculate TonCuoiKy in the query
	countQuery := `
		SELECT COUNT(*) FROM supplies 
		WHERE (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) < ?
	`
	var total int
	err := r.DB.QueryRow(countQuery, threshold).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting low stock supplies: %w", err)
	}

	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT, QUY_CACH_DONG_GOI AS QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) < ?
		ORDER BY (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, threshold, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying low stock supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		var s Supply
		err := rows.Scan(
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2, &s.MaHieu,
			&s.TypeName, &s.Name, &s.Unit, &s.QuyCach, &s.ThongTinThau, &s.TongThau,
			&s.HangSX, &s.NuocSX, &s.NhaCungCap, &s.Price,
			&s.TonDauKy, &s.NhapTrongKy, &s.XuatTrongKy, &s.TongNhap,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}

		s.TonCuoiKy = calculateTonCuoiKy(s.TonDauKy, s.NhapTrongKy, s.XuatTrongKy)
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

// GetCompareCatalog retrieves comparison catalog rows with pagination, keyword search, and ma_thong_tu_04 level filtering.
func (r *SupplyRepository) GetCompareCatalog(keyword string, level1Filter string, level2Filter string, page, pageSize int) ([]CompareSupply, int, error) {
	offset := (page - 1) * pageSize
	search := "%" + keyword + "%"
	level1Search := strings.TrimSpace(level1Filter)
	level2Search := strings.TrimSpace(level2Filter)

	countQuery := `
		SELECT COUNT(*)
		FROM so_sanh_vat_tu
		WHERE (? = '' OR ma_thu_vien LIKE ? OR ten_vat_tu LIKE ? OR ten_cong_ty LIKE ? OR ma_thong_tu_04 LIKE ?)
		  AND (? = '' OR (
			LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) > 4
			AND SUBSTRING(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 3, 1) = '.'
			AND LEFT(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 4) = ?
		  ))
		  AND (? = '' OR RIGHT(TRIM(IFNULL(ma_thong_tu_04, '')), 3) = ?)
	`

	var total int
	err := r.DB.QueryRow(
		countQuery,
		keyword,
		search,
		search,
		search,
		search,
		level1Search,
		level1Search,
		level2Search,
		level2Search,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting compare catalog: %w", err)
	}

	query := `
		SELECT
			stt, ten_cong_ty, ma_thu_vien, ma_thong_tu_04, ten_vat_tu,
			ten_thuong_mai, tskt_2025, tskt_2026, chat_lieu_vat_lieu,
			dac_tinh_cau_tao, kich_thuoc, chieu_dai, tinh_nang_su_dung, tskt_khac, dvt,
			so_luong_su_dung_12_thang, so_luong_trung_thau_2025_bo_sung,
			don_gia_trung_thau_2025, don_gia_de_xuat_2026,
			ket_qua_trung_thau_thap_nhat, thoi_gian_don_vi_dang_tai_thap_nhat,
			ket_qua_trung_thau_cao_nhat, thoi_gian_don_vi_dang_tai_cao_nhat,
			ma_so_thue, ma_hieu, hangsx, nuoc_sx, nhom_nuoc, chat_luong,
			ma_5086, created_at, updated_at
		FROM so_sanh_vat_tu
		WHERE (? = '' OR ma_thu_vien LIKE ? OR ten_vat_tu LIKE ? OR ten_cong_ty LIKE ? OR ma_thong_tu_04 LIKE ?)
		  AND (? = '' OR (
			LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) > 4
			AND SUBSTRING(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 3, 1) = '.'
			AND LEFT(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 4) = ?
		  ))
		  AND (? = '' OR RIGHT(TRIM(IFNULL(ma_thong_tu_04, '')), 3) = ?)
		ORDER BY stt
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(
		query,
		keyword,
		search,
		search,
		search,
		search,
		level1Search,
		level1Search,
		level2Search,
		level2Search,
		pageSize,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying compare catalog: %w", err)
	}
	defer rows.Close()

	items := []CompareSupply{}
	for rows.Next() {
		item, scanErr := scanCompareSupplyRow(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("error scanning compare row: %w", scanErr)
		}
		items = append(items, item)
	}

	return items, total, nil
}

// GetCompareLevel1Options retrieves distinct level 1 values from ma_thong_tu_04.
func (r *SupplyRepository) GetCompareLevel1Options() ([]string, error) {
	query := `
		SELECT DISTINCT LEFT(TRIM(ma_thong_tu_04), LENGTH(TRIM(ma_thong_tu_04)) - 4) AS level1
		FROM so_sanh_vat_tu
		WHERE LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) > 4
		  AND SUBSTRING(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 3, 1) = '.'
		ORDER BY level1
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying compare groups: %w", err)
	}
	defer rows.Close()

	groups := []string{}
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("error scanning compare level1 option: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// GetCompareLevel2Options retrieves distinct level 2 values (last 3 chars) for a selected level 1.
func (r *SupplyRepository) GetCompareLevel2Options(level1 string) ([]string, error) {
	level1Search := strings.TrimSpace(level1)

	query := `
		SELECT DISTINCT RIGHT(TRIM(ma_thong_tu_04), 3) AS level2
		FROM so_sanh_vat_tu
		WHERE LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) > 4
		  AND SUBSTRING(TRIM(IFNULL(ma_thong_tu_04, '')), LENGTH(TRIM(IFNULL(ma_thong_tu_04, ''))) - 3, 1) = '.'
		  AND (? = '' OR LEFT(TRIM(ma_thong_tu_04), LENGTH(TRIM(ma_thong_tu_04)) - 4) = ?)
		ORDER BY level2
	`

	rows, err := r.DB.Query(query, level1Search, level1Search)
	if err != nil {
		return nil, fmt.Errorf("error querying compare level2 options: %w", err)
	}
	defer rows.Close()

	level2Options := []string{}
	for rows.Next() {
		var level2 string
		if err := rows.Scan(&level2); err != nil {
			return nil, fmt.Errorf("error scanning compare level2 option: %w", err)
		}
		level2Options = append(level2Options, level2)
	}

	return level2Options, nil
}

// GetCompareByLibraryCodes retrieves comparison rows for selected library codes.
func (r *SupplyRepository) GetCompareByLibraryCodes(maThuVien []string) ([]CompareSupply, error) {
	if len(maThuVien) == 0 {
		return []CompareSupply{}, nil
	}

	placeholder := strings.TrimRight(strings.Repeat("?,", len(maThuVien)), ",")

	query := fmt.Sprintf(`
		SELECT
			stt, ten_cong_ty, ma_thu_vien, ma_thong_tu_04, ten_vat_tu,
			ten_thuong_mai, tskt_2025, tskt_2026, chat_lieu_vat_lieu,
			dac_tinh_cau_tao, kich_thuoc, chieu_dai, tinh_nang_su_dung, tskt_khac, dvt,
			so_luong_su_dung_12_thang, so_luong_trung_thau_2025_bo_sung,
			don_gia_trung_thau_2025, don_gia_de_xuat_2026,
			ket_qua_trung_thau_thap_nhat, thoi_gian_don_vi_dang_tai_thap_nhat,
			ket_qua_trung_thau_cao_nhat, thoi_gian_don_vi_dang_tai_cao_nhat,
			ma_so_thue, ma_hieu, hangsx, nuoc_sx, nhom_nuoc, chat_luong,
			ma_5086, created_at, updated_at
		FROM so_sanh_vat_tu
		WHERE ma_thu_vien IN (%s)
		ORDER BY FIELD(ma_thu_vien, %s)
	`, placeholder, placeholder)

	args := make([]interface{}, 0, len(maThuVien)*2)
	for _, code := range maThuVien {
		args = append(args, code)
	}
	for _, code := range maThuVien {
		args = append(args, code)
	}

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying compare supplies: %w", err)
	}
	defer rows.Close()

	items := []CompareSupply{}
	for rows.Next() {
		item, scanErr := scanCompareSupplyRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("error scanning compare row: %w", scanErr)
		}
		items = append(items, item)
	}

	return items, nil
}
