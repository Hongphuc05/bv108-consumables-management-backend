package models

import (
	"database/sql"
	"time"
)

// HoaDon represents an invoice record from UBot
type HoaDon struct {
	ID               int       `json:"id"`
	TrangThaiHoaDon  string    `json:"trangThaiHoaDon"`
	LoaiHoaDon       string    `json:"loaiHoaDon"`
	SoHoaDon         string    `json:"soHoaDon"`
	NgayHoaDon       time.Time `json:"ngayHoaDon"`
	MaSoThueNguoiBan string    `json:"maSoThueNguoiBan"`
	CongTy           string    `json:"congTy"`
	DiaChi           string    `json:"diaChi"`
	LinkTraCuuHoaDon string    `json:"linkTraCuuHoaDon"`
	IDHoaDon         string    `json:"idHoaDon"`
	STTDongHang      int       `json:"sttDongHang"`
	TenHangHoa       string    `json:"tenHangHoa"`
	MaHangHoa        string    `json:"maHangHoa"`
	DonViTinh        string    `json:"donViTinh"`
	SoLuong          float64   `json:"soLuong"`
	DonGiaChuaThue   float64   `json:"donGiaChuaThue"`
	ThueSuatGTGT     float64   `json:"thueSuatGtgt"`
}

// HoaDonRepository handles database operations for hoa_don table
type HoaDonRepository struct {
	db *sql.DB
}

// NewHoaDonRepository creates a new repository instance
func NewHoaDonRepository(db *sql.DB) *HoaDonRepository {
	return &HoaDonRepository{db: db}
}

// GetAll retrieves all invoices from database
func (r *HoaDonRepository) GetAll(limit, offset int) ([]HoaDon, error) {
	query := `
		SELECT 
			id, trang_thai_hoa_don, loai_hoa_don, so_hoa_don, ngay_hoa_don,
			ma_so_thue_nguoi_ban, cong_ty, dia_chi, link_tra_cuu_hoa_don,
			id_hoa_don, stt_dong_hang, ten_hang_hoa, ma_hang_hoa,
			don_vi_tinh, so_luong, don_gia_chua_thue, thue_suat_gtgt
		FROM hoa_don
		ORDER BY ngay_hoa_don DESC, id DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hoaDons []HoaDon
	for rows.Next() {
		var hd HoaDon
		err := rows.Scan(
			&hd.ID, &hd.TrangThaiHoaDon, &hd.LoaiHoaDon, &hd.SoHoaDon, &hd.NgayHoaDon,
			&hd.MaSoThueNguoiBan, &hd.CongTy, &hd.DiaChi, &hd.LinkTraCuuHoaDon,
			&hd.IDHoaDon, &hd.STTDongHang, &hd.TenHangHoa, &hd.MaHangHoa,
			&hd.DonViTinh, &hd.SoLuong, &hd.DonGiaChuaThue, &hd.ThueSuatGTGT,
		)
		if err != nil {
			return nil, err
		}
		hoaDons = append(hoaDons, hd)
	}

	return hoaDons, nil
}

// GetCount returns total number of invoice records
func (r *HoaDonRepository) GetCount() (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM hoa_don"
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}

// GetByIDHoaDon retrieves all line items for a specific invoice ID
func (r *HoaDonRepository) GetByIDHoaDon(idHoaDon string) ([]HoaDon, error) {
	query := `
		SELECT 
			id, trang_thai_hoa_don, loai_hoa_don, so_hoa_don, ngay_hoa_don,
			ma_so_thue_nguoi_ban, cong_ty, dia_chi, link_tra_cuu_hoa_don,
			id_hoa_don, stt_dong_hang, ten_hang_hoa, ma_hang_hoa,
			don_vi_tinh, so_luong, don_gia_chua_thue, thue_suat_gtgt
		FROM hoa_don
		WHERE id_hoa_don = ?
		ORDER BY stt_dong_hang
	`

	rows, err := r.db.Query(query, idHoaDon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hoaDons []HoaDon
	for rows.Next() {
		var hd HoaDon
		err := rows.Scan(
			&hd.ID, &hd.TrangThaiHoaDon, &hd.LoaiHoaDon, &hd.SoHoaDon, &hd.NgayHoaDon,
			&hd.MaSoThueNguoiBan, &hd.CongTy, &hd.DiaChi, &hd.LinkTraCuuHoaDon,
			&hd.IDHoaDon, &hd.STTDongHang, &hd.TenHangHoa, &hd.MaHangHoa,
			&hd.DonViTinh, &hd.SoLuong, &hd.DonGiaChuaThue, &hd.ThueSuatGTGT,
		)
		if err != nil {
			return nil, err
		}
		hoaDons = append(hoaDons, hd)
	}

	return hoaDons, nil
}

// SearchByKeyword searches invoices by keyword in various fields
func (r *HoaDonRepository) SearchByKeyword(keyword string, limit, offset int) ([]HoaDon, error) {
	query := `
		SELECT 
			id, trang_thai_hoa_don, loai_hoa_don, so_hoa_don, ngay_hoa_don,
			ma_so_thue_nguoi_ban, cong_ty, dia_chi, link_tra_cuu_hoa_don,
			id_hoa_don, stt_dong_hang, ten_hang_hoa, ma_hang_hoa,
			don_vi_tinh, so_luong, don_gia_chua_thue, thue_suat_gtgt
		FROM hoa_don
		WHERE 
			so_hoa_don LIKE ? OR
			cong_ty LIKE ? OR
			ten_hang_hoa LIKE ? OR
			ma_hang_hoa LIKE ?
		ORDER BY ngay_hoa_don DESC, id DESC
		LIMIT ? OFFSET ?
	`

	searchTerm := "%" + keyword + "%"
	rows, err := r.db.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hoaDons []HoaDon
	for rows.Next() {
		var hd HoaDon
		err := rows.Scan(
			&hd.ID, &hd.TrangThaiHoaDon, &hd.LoaiHoaDon, &hd.SoHoaDon, &hd.NgayHoaDon,
			&hd.MaSoThueNguoiBan, &hd.CongTy, &hd.DiaChi, &hd.LinkTraCuuHoaDon,
			&hd.IDHoaDon, &hd.STTDongHang, &hd.TenHangHoa, &hd.MaHangHoa,
			&hd.DonViTinh, &hd.SoLuong, &hd.DonGiaChuaThue, &hd.ThueSuatGTGT,
		)
		if err != nil {
			return nil, err
		}
		hoaDons = append(hoaDons, hd)
	}

	return hoaDons, nil
}
