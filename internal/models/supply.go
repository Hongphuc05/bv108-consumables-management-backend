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
	TenVatTu2025               sql.NullString  `json:"tenVatTu2025"`
	ThongSoMoiThau2025         sql.NullString  `json:"thongSoMoiThau2025"`
	ThongSoHieuChinh2026       sql.NullString  `json:"thongSoHieuChinh2026"`
	ThongSoKyThuat1            sql.NullString  `json:"thongSoKyThuat1"`
	ThongSoKyThuat2            sql.NullString  `json:"thongSoKyThuat2"`
	ThongSoKyThuat3            sql.NullString  `json:"thongSoKyThuat3"`
	ThongSoKyThuat4            sql.NullString  `json:"thongSoKyThuat4"`
	ThongSoKyThuat5            sql.NullString  `json:"thongSoKyThuat5"`
	ThongSoKyThuat9            sql.NullString  `json:"thongSoKyThuat9"`
	MaVtthTuongDuong           sql.NullString  `json:"maVtthTuongDuong"`
	CongTyVtthTuongDuong       sql.NullString  `json:"congTyVtthTuongDuong"`
	DVT                        sql.NullString  `json:"dvt"`
	SoLuongSuDung12Thang       sql.NullFloat64 `json:"soLuongSuDung12Thang"`
	SoLuongTrungThau2025BoSung sql.NullInt32   `json:"soLuongTrungThau2025BoSung"`
	DonGiaTrungThau2025        sql.NullFloat64 `json:"donGiaTrungThau2025"`
	DonGiaDeXuat2026           sql.NullFloat64 `json:"donGiaDeXuat2026"`
	KetQuaTrungThauThapNhat    sql.NullFloat64 `json:"ketQuaTrungThauThapNhat"`
	ThoiGianDangTaiThapNhat    sql.NullString  `json:"thoiGianDangTaiThapNhat"`
	KetQuaTrungThauCaoNhat     sql.NullFloat64 `json:"ketQuaTrungThauCaoNhat"`
	ThoiGianDangTaiCaoNhat     sql.NullString  `json:"thoiGianDangTaiCaoNhat"`
	CongTyThamKhao             sql.NullString  `json:"congTyThamKhao"`
	MaSoThue                   sql.NullString  `json:"maSoThue"`
	KyMaHieu                   sql.NullString  `json:"kyMaHieu"`
	HangSanXuat                sql.NullString  `json:"hangSanXuat"`
	NuocSanXuat                sql.NullString  `json:"nuocSanXuat"`
	NhomNuoc                   sql.NullString  `json:"nhomNuoc"`
	ChatLuong                  sql.NullString  `json:"chatLuong"`
	Ma5086                     sql.NullString  `json:"ma5086"`
	TenThuongMai               sql.NullString  `json:"tenThuongMai"`
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
		&item.TenVatTu2025,
		&item.ThongSoMoiThau2025,
		&item.ThongSoHieuChinh2026,
		&item.ThongSoKyThuat1,
		&item.ThongSoKyThuat2,
		&item.ThongSoKyThuat3,
		&item.ThongSoKyThuat4,
		&item.ThongSoKyThuat5,
		&item.ThongSoKyThuat9,
		&item.MaVtthTuongDuong,
		&item.CongTyVtthTuongDuong,
		&item.DVT,
		&item.SoLuongSuDung12Thang,
		&item.SoLuongTrungThau2025BoSung,
		&item.DonGiaTrungThau2025,
		&item.DonGiaDeXuat2026,
		&item.KetQuaTrungThauThapNhat,
		&item.ThoiGianDangTaiThapNhat,
		&item.KetQuaTrungThauCaoNhat,
		&item.ThoiGianDangTaiCaoNhat,
		&item.CongTyThamKhao,
		&item.MaSoThue,
		&item.KyMaHieu,
		&item.HangSanXuat,
		&item.NuocSanXuat,
		&item.NhomNuoc,
		&item.ChatLuong,
		&item.Ma5086,
		&item.TenThuongMai,
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
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, TYPENAME, NAME, UNIT, QUY_CACH,
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
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2,
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
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, TYPENAME, NAME, UNIT, QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE IDX1 = ?
	`

	var s Supply
	err := r.DB.QueryRow(query, idx1).Scan(
		&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2,
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
	countQuery := "SELECT COUNT(*) FROM supplies WHERE NAME LIKE ? OR ID LIKE ?"
	err := r.DB.QueryRow(countQuery, searchPattern, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	// Get paginated data
	query := `
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, TYPENAME, NAME, UNIT, QUY_CACH,
			THONG_TIN_THAU, TONGTHAU, HANGSX, NUOC_SX, NHA_CUNG_CAP,
			PRICE, TONDAUKY, NHAPTRONGKY, XUATTRONGKY, TONGNHAP
		FROM supplies
		WHERE NAME LIKE ? OR ID LIKE ?
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, searchPattern, searchPattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		var s Supply
		err := rows.Scan(
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2,
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
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, TYPENAME, NAME, UNIT, QUY_CACH,
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
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2,
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
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, TYPENAME, NAME, UNIT, QUY_CACH,
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
			&s.IDX1, &s.ProductID, &s.GroupName, &s.ID, &s.IDX2,
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

// GetCompareCatalog retrieves comparison catalog rows with pagination and keyword search.
func (r *SupplyRepository) GetCompareCatalog(keyword string, page, pageSize int) ([]CompareSupply, int, error) {
	offset := (page - 1) * pageSize
	search := "%" + keyword + "%"

	countQuery := `
		SELECT COUNT(*)
		FROM so_sanh_vat_tu
		WHERE (? = '' OR ma_thu_vien LIKE ? OR ten_vat_tu_2025 LIKE ? OR ten_cong_ty LIKE ?)
	`

	var total int
	err := r.DB.QueryRow(countQuery, keyword, search, search, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting compare catalog: %w", err)
	}

	query := `
		SELECT
			stt, ten_cong_ty, ma_thu_vien, ma_thong_tu_04, ten_vat_tu_2025,
			thong_so_moi_thau_2025, thong_so_hieu_chinh_2026,
			thong_so_ky_thuat_1, thong_so_ky_thuat_2, thong_so_ky_thuat_3,
			thong_so_ky_thuat_4, thong_so_ky_thuat_5, thong_so_ky_thuat_9,
			ma_vtth_tuong_duong, cong_ty_vtth_tuong_duong, dvt,
			so_luong_su_dung_12_thang, so_luong_trung_thau_2025_bo_sung,
			don_gia_trung_thau_2025, don_gia_de_xuat_2026,
			ket_qua_trung_thau_thap_nhat, thoi_gian_don_vi_dang_tai_thap_nhat,
			ket_qua_trung_thau_cao_nhat, thoi_gian_don_vi_dang_tai_cao_nhat,
			cong_ty_tham_khao, ma_so_thue, ky_ma_hieu,
			hang_san_xuat, nuoc_san_xuat, nhom_nuoc, chat_luong,
			ma_5086, ten_thuong_mai, created_at, updated_at
		FROM so_sanh_vat_tu
		WHERE (? = '' OR ma_thu_vien LIKE ? OR ten_vat_tu_2025 LIKE ? OR ten_cong_ty LIKE ?)
		ORDER BY stt
		LIMIT ? OFFSET ?
	`

	rows, err := r.DB.Query(query, keyword, search, search, search, pageSize, offset)
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

// GetCompareByLibraryCodes retrieves comparison rows for selected library codes.
func (r *SupplyRepository) GetCompareByLibraryCodes(maThuVien []string) ([]CompareSupply, error) {
	if len(maThuVien) == 0 {
		return []CompareSupply{}, nil
	}

	placeholder := strings.TrimRight(strings.Repeat("?,", len(maThuVien)), ",")

	query := fmt.Sprintf(`
		SELECT
			stt, ten_cong_ty, ma_thu_vien, ma_thong_tu_04, ten_vat_tu_2025,
			thong_so_moi_thau_2025, thong_so_hieu_chinh_2026,
			thong_so_ky_thuat_1, thong_so_ky_thuat_2, thong_so_ky_thuat_3,
			thong_so_ky_thuat_4, thong_so_ky_thuat_5, thong_so_ky_thuat_9,
			ma_vtth_tuong_duong, cong_ty_vtth_tuong_duong, dvt,
			so_luong_su_dung_12_thang, so_luong_trung_thau_2025_bo_sung,
			don_gia_trung_thau_2025, don_gia_de_xuat_2026,
			ket_qua_trung_thau_thap_nhat, thoi_gian_don_vi_dang_tai_thap_nhat,
			ket_qua_trung_thau_cao_nhat, thoi_gian_don_vi_dang_tai_cao_nhat,
			cong_ty_tham_khao, ma_so_thue, ky_ma_hieu,
			hang_san_xuat, nuoc_san_xuat, nhom_nuoc, chat_luong,
			ma_5086, ten_thuong_mai, created_at, updated_at
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
