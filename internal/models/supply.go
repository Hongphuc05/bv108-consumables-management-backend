package models

import (
	"database/sql"
	"fmt"
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
