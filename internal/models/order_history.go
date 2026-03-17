package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	OrderSourceForecast = "forecast"
	OrderSourceManual   = "manual"
)

type OrderActor struct {
	ID       int64
	Username string
	Email    string
}

type PendingOrder struct {
	ID               int64  `json:"id"`
	CompanyContactID *int64 `json:"companyContactId,omitempty"`
	NhaThau          string `json:"nhaThau"`
	MaQuanLy         string `json:"maQuanLy"`
	MaVtytCu         string `json:"maVtytCu"`
	TenVtytBv        string `json:"tenVtytBv"`
	MaHieu           string `json:"maHieu"`
	HangSx           string `json:"hangSx"`
	DonViTinh        string `json:"donViTinh"`
	QuyCach          string `json:"quyCach"`
	DotGoiHang       int    `json:"dotGoiHang"`
	Email            string `json:"email,omitempty"`
	Source           string `json:"source"`
	NguoiPheDuyet    string `json:"nguoiPheDuyet,omitempty"`
	NguoiPheDuyetEmail string `json:"nguoiPheDuyetEmail,omitempty"`
	ThoiGianPheDuyet string `json:"thoiGianPheDuyet,omitempty"`
	NguoiTaoDon      string `json:"nguoiTaoDon,omitempty"`
	NguoiTaoDonEmail string `json:"nguoiTaoDonEmail,omitempty"`
	NgayTao          string `json:"ngayTao,omitempty"`
}

type OrderHistoryRecord struct {
	PendingOrder
	NgayDatHang        string `json:"ngayDatHang"`
	TrangThai          string `json:"trangThai"`
	EmailSent          bool   `json:"emailSent"`
	NguoiDatHang       string `json:"nguoiDatHang"`
	NguoiDatHangEmail  string `json:"nguoiDatHangEmail,omitempty"`
}

type CreatePendingOrderInput struct {
	NhaThau          string
	MaQuanLy         string
	MaVtytCu         string
	TenVtytBv        string
	MaHieu           string
	HangSx           string
	DonViTinh        string
	QuyCach          string
	DotGoiHang       int
	Email            string
	Source           string
	Approver         *OrderActor
	CreatedBy        OrderActor
	ApprovalTime     string
}

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) EnsureSchema() error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS pending_orders (
			id BIGINT NOT NULL AUTO_INCREMENT,
			company_contact_id BIGINT NULL,
			nha_thau VARCHAR(255) NOT NULL,
			ma_quan_ly VARCHAR(255) NOT NULL DEFAULT '',
			ma_vtyt_cu VARCHAR(255) NOT NULL,
			ten_vtyt_bv VARCHAR(500) NOT NULL,
			ma_hieu VARCHAR(255) NOT NULL DEFAULT '',
			hang_sx VARCHAR(255) NOT NULL DEFAULT '',
			don_vi_tinh VARCHAR(100) NOT NULL DEFAULT '',
			quy_cach VARCHAR(255) NOT NULL DEFAULT '',
			so_luong INT NOT NULL,
			email VARCHAR(255) NOT NULL DEFAULT '',
			source VARCHAR(50) NOT NULL,
			nguoi_phe_duyet_id BIGINT NULL,
			nguoi_phe_duyet VARCHAR(255) NOT NULL DEFAULT '',
			nguoi_phe_duyet_email VARCHAR(255) NOT NULL DEFAULT '',
			thoi_gian_phe_duyet VARCHAR(64) NOT NULL DEFAULT '',
			nguoi_tao_don_id BIGINT NULL,
			nguoi_tao_don VARCHAR(255) NOT NULL DEFAULT '',
			nguoi_tao_don_email VARCHAR(255) NOT NULL DEFAULT '',
			ngay_tao VARCHAR(64) NOT NULL,
			updated_at VARCHAR(64) NOT NULL,
			PRIMARY KEY (id),
			KEY idx_pending_orders_company_contact (company_contact_id),
			KEY idx_pending_orders_created_at (updated_at, id),
			KEY idx_pending_orders_source_code (source, ma_vtyt_cu)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
		`
		CREATE TABLE IF NOT EXISTS order_history (
			id BIGINT NOT NULL AUTO_INCREMENT,
			pending_order_id BIGINT NULL,
			company_contact_id BIGINT NULL,
			nha_thau VARCHAR(255) NOT NULL,
			ma_quan_ly VARCHAR(255) NOT NULL DEFAULT '',
			ma_vtyt_cu VARCHAR(255) NOT NULL,
			ten_vtyt_bv VARCHAR(500) NOT NULL,
			ma_hieu VARCHAR(255) NOT NULL DEFAULT '',
			hang_sx VARCHAR(255) NOT NULL DEFAULT '',
			don_vi_tinh VARCHAR(100) NOT NULL DEFAULT '',
			quy_cach VARCHAR(255) NOT NULL DEFAULT '',
			so_luong INT NOT NULL,
			email VARCHAR(255) NOT NULL DEFAULT '',
			source VARCHAR(50) NOT NULL,
			nguoi_phe_duyet_id BIGINT NULL,
			nguoi_phe_duyet VARCHAR(255) NOT NULL DEFAULT '',
			nguoi_phe_duyet_email VARCHAR(255) NOT NULL DEFAULT '',
			thoi_gian_phe_duyet VARCHAR(64) NOT NULL DEFAULT '',
			nguoi_tao_don_id BIGINT NULL,
			nguoi_tao_don VARCHAR(255) NOT NULL DEFAULT '',
			nguoi_tao_don_email VARCHAR(255) NOT NULL DEFAULT '',
			ngay_tao VARCHAR(64) NOT NULL DEFAULT '',
			ngay_dat_hang VARCHAR(64) NOT NULL,
			trang_thai VARCHAR(100) NOT NULL,
			email_sent TINYINT(1) NOT NULL DEFAULT 0,
			nguoi_dat_hang_id BIGINT NOT NULL,
			nguoi_dat_hang VARCHAR(255) NOT NULL,
			nguoi_dat_hang_email VARCHAR(255) NOT NULL DEFAULT '',
			PRIMARY KEY (id),
			KEY idx_order_history_company_contact (company_contact_id),
			KEY idx_order_history_ngay_dat_hang (ngay_dat_hang, id),
			KEY idx_order_history_ma_quan_ly (ma_quan_ly)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error ensuring order schema: %w", err)
		}
	}

	if err := r.ensureQuantityColumn("pending_orders"); err != nil {
		return err
	}

	if err := r.ensureQuantityColumn("order_history"); err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) ListPendingOrders() ([]PendingOrder, error) {
	rows, err := r.DB.Query(`
		SELECT
			id,
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ma_hieu,
			hang_sx,
			don_vi_tinh,
			quy_cach,
			so_luong,
			email,
			source,
			nguoi_phe_duyet,
			nguoi_phe_duyet_email,
			thoi_gian_phe_duyet,
			nguoi_tao_don,
			nguoi_tao_don_email,
			ngay_tao
		FROM pending_orders
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing pending orders: %w", err)
	}
	defer rows.Close()

	orders := make([]PendingOrder, 0)
	for rows.Next() {
		var order PendingOrder
		var companyContactID sql.NullInt64
		if err := rows.Scan(
			&order.ID,
			&companyContactID,
			&order.NhaThau,
			&order.MaQuanLy,
			&order.MaVtytCu,
			&order.TenVtytBv,
			&order.MaHieu,
			&order.HangSx,
			&order.DonViTinh,
			&order.QuyCach,
			&order.DotGoiHang,
			&order.Email,
			&order.Source,
			&order.NguoiPheDuyet,
			&order.NguoiPheDuyetEmail,
			&order.ThoiGianPheDuyet,
			&order.NguoiTaoDon,
			&order.NguoiTaoDonEmail,
			&order.NgayTao,
		); err != nil {
			return nil, fmt.Errorf("error scanning pending order: %w", err)
		}
		if companyContactID.Valid {
			value := companyContactID.Int64
			order.CompanyContactID = &value
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending orders: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) GetPendingOrdersByIDs(orderIDs []int64) ([]PendingOrder, error) {
	if len(orderIDs) == 0 {
		return []PendingOrder{}, nil
	}

	placeholders := makePlaceholders(len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for index, orderID := range orderIDs {
		args[index] = orderID
	}

	rows, err := r.DB.Query(fmt.Sprintf(`
		SELECT
			id,
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ma_hieu,
			hang_sx,
			don_vi_tinh,
			quy_cach,
			so_luong,
			email,
			source,
			nguoi_phe_duyet,
			nguoi_phe_duyet_email,
			thoi_gian_phe_duyet,
			nguoi_tao_don,
			nguoi_tao_don_email,
			ngay_tao
		FROM pending_orders
		WHERE id IN (%s)
		ORDER BY updated_at DESC, id DESC
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("error listing pending orders by ids: %w", err)
	}
	defer rows.Close()

	orders := make([]PendingOrder, 0, len(orderIDs))
	for rows.Next() {
		var order PendingOrder
		var companyContactID sql.NullInt64
		if err := rows.Scan(
			&order.ID,
			&companyContactID,
			&order.NhaThau,
			&order.MaQuanLy,
			&order.MaVtytCu,
			&order.TenVtytBv,
			&order.MaHieu,
			&order.HangSx,
			&order.DonViTinh,
			&order.QuyCach,
			&order.DotGoiHang,
			&order.Email,
			&order.Source,
			&order.NguoiPheDuyet,
			&order.NguoiPheDuyetEmail,
			&order.ThoiGianPheDuyet,
			&order.NguoiTaoDon,
			&order.NguoiTaoDonEmail,
			&order.NgayTao,
		); err != nil {
			return nil, fmt.Errorf("error scanning pending order by id: %w", err)
		}
		if companyContactID.Valid {
			value := companyContactID.Int64
			order.CompanyContactID = &value
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending orders by ids: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) ListOrderHistory() ([]OrderHistoryRecord, error) {
	rows, err := r.DB.Query(`
		SELECT
			id,
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ma_hieu,
			hang_sx,
			don_vi_tinh,
			quy_cach,
			so_luong,
			email,
			source,
			nguoi_phe_duyet,
			nguoi_phe_duyet_email,
			thoi_gian_phe_duyet,
			nguoi_tao_don,
			nguoi_tao_don_email,
			ngay_tao,
			ngay_dat_hang,
			trang_thai,
			email_sent,
			nguoi_dat_hang,
			nguoi_dat_hang_email
		FROM order_history
		ORDER BY ngay_dat_hang DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing order history: %w", err)
	}
	defer rows.Close()

	history := make([]OrderHistoryRecord, 0)
	for rows.Next() {
		var item OrderHistoryRecord
		var companyContactID sql.NullInt64
		var emailSent int
		if err := rows.Scan(
			&item.ID,
			&companyContactID,
			&item.NhaThau,
			&item.MaQuanLy,
			&item.MaVtytCu,
			&item.TenVtytBv,
			&item.MaHieu,
			&item.HangSx,
			&item.DonViTinh,
			&item.QuyCach,
			&item.DotGoiHang,
			&item.Email,
			&item.Source,
			&item.NguoiPheDuyet,
			&item.NguoiPheDuyetEmail,
			&item.ThoiGianPheDuyet,
			&item.NguoiTaoDon,
			&item.NguoiTaoDonEmail,
			&item.NgayTao,
			&item.NgayDatHang,
			&item.TrangThai,
			&emailSent,
			&item.NguoiDatHang,
			&item.NguoiDatHangEmail,
		); err != nil {
			return nil, fmt.Errorf("error scanning order history: %w", err)
		}
		if companyContactID.Valid {
			value := companyContactID.Int64
			item.CompanyContactID = &value
		}
		item.EmailSent = emailSent == 1
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order history: %w", err)
	}

	return history, nil
}

func (r *OrderRepository) AddForecastOrders(inputs []CreatePendingOrderInput) error {
	if len(inputs) == 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting forecast order transaction: %w", err)
	}
	defer tx.Rollback()

	for _, input := range inputs {
		now := currentTimestamp()
		contactRepo := NewCompanyContactRepository(r.DB)
		companyContactID, resolvedEmail, err := contactRepo.EnsureContactTx(tx, input.NhaThau, "", input.Email)
		if err != nil {
			return fmt.Errorf("error resolving company contact: %w", err)
		}

		var existingID int64
		var existingQty int
		err = tx.QueryRow(`
			SELECT id, so_luong
			FROM pending_orders
			WHERE source = ? AND ma_vtyt_cu = ?
			LIMIT 1
		`, OrderSourceForecast, input.MaVtytCu).Scan(&existingID, &existingQty)

		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("error checking existing forecast order: %w", err)
		}

		if err == sql.ErrNoRows {
			if err := r.insertPendingOrderTx(tx, input, now); err != nil {
				return err
			}
			continue
		}

		updatedQty := existingQty + input.DotGoiHang
		approverID := nullableInt64(input.Approver)
		approverName := nullableActorField(input.Approver, func(actor *OrderActor) string { return actor.Username })
		approverEmail := nullableActorField(input.Approver, func(actor *OrderActor) string { return actor.Email })

		if _, err := tx.Exec(`
			UPDATE pending_orders
			SET
				nha_thau = ?,
				ma_quan_ly = ?,
				ten_vtyt_bv = ?,
				ma_hieu = ?,
				hang_sx = ?,
				don_vi_tinh = ?,
				quy_cach = ?,
				so_luong = ?,
				company_contact_id = ?,
				email = ?,
				nguoi_phe_duyet_id = ?,
				nguoi_phe_duyet = ?,
				nguoi_phe_duyet_email = ?,
				thoi_gian_phe_duyet = ?,
				nguoi_tao_don_id = ?,
				nguoi_tao_don = ?,
				nguoi_tao_don_email = ?,
				updated_at = ?
			WHERE id = ?
		`,
			input.NhaThau,
			input.MaQuanLy,
			input.TenVtytBv,
			input.MaHieu,
			input.HangSx,
			input.DonViTinh,
			input.QuyCach,
			updatedQty,
			nullInt64ToValue(companyContactID),
			resolvedEmail,
			approverID,
			approverName,
			approverEmail,
			input.ApprovalTime,
			input.CreatedBy.ID,
			input.CreatedBy.Username,
			input.CreatedBy.Email,
			now,
			existingID,
		); err != nil {
			return fmt.Errorf("error updating forecast order: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing forecast orders: %w", err)
	}

	return nil
}

func (r *OrderRepository) AddManualOrder(input CreatePendingOrderInput) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting manual order transaction: %w", err)
	}
	defer tx.Rollback()

	if err := r.insertPendingOrderTx(tx, input, currentTimestamp()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing manual order: %w", err)
	}

	return nil
}

func (r *OrderRepository) PlaceOrders(orderIDs []int64, placedBy OrderActor) (int, error) {
	if len(orderIDs) == 0 {
		return 0, nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting place order transaction: %w", err)
	}
	defer tx.Rollback()

	placeholders := makePlaceholders(len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for index, orderID := range orderIDs {
		args[index] = orderID
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ma_hieu,
			hang_sx,
			don_vi_tinh,
			quy_cach,
			so_luong,
			email,
			source,
			nguoi_phe_duyet_id,
			nguoi_phe_duyet,
			nguoi_phe_duyet_email,
			thoi_gian_phe_duyet,
			nguoi_tao_don_id,
			nguoi_tao_don,
			nguoi_tao_don_email,
			ngay_tao
		FROM pending_orders
		WHERE id IN (%s)
		ORDER BY updated_at DESC, id DESC
	`, placeholders)

	rows, err := tx.Query(query, args...)
	if err != nil {
		return 0, fmt.Errorf("error selecting pending orders to place: %w", err)
	}
	defer rows.Close()

	type pendingOrderRow struct {
		ID                  int64
		CompanyContactID    sql.NullInt64
		NhaThau             string
		MaQuanLy            string
		MaVtytCu            string
		TenVtytBv           string
		MaHieu              string
		HangSx              string
		DonViTinh           string
		QuyCach             string
		DotGoiHang          int
		Email               string
		Source              string
		NguoiPheDuyetID     sql.NullInt64
		NguoiPheDuyet       string
		NguoiPheDuyetEmail  string
		ThoiGianPheDuyet    string
		NguoiTaoDonID       sql.NullInt64
		NguoiTaoDon         string
		NguoiTaoDonEmail    string
		NgayTao             string
	}

	selectedOrders := make([]pendingOrderRow, 0, len(orderIDs))
	for rows.Next() {
		var order pendingOrderRow
		if err := rows.Scan(
			&order.ID,
			&order.CompanyContactID,
			&order.NhaThau,
			&order.MaQuanLy,
			&order.MaVtytCu,
			&order.TenVtytBv,
			&order.MaHieu,
			&order.HangSx,
			&order.DonViTinh,
			&order.QuyCach,
			&order.DotGoiHang,
			&order.Email,
			&order.Source,
			&order.NguoiPheDuyetID,
			&order.NguoiPheDuyet,
			&order.NguoiPheDuyetEmail,
			&order.ThoiGianPheDuyet,
			&order.NguoiTaoDonID,
			&order.NguoiTaoDon,
			&order.NguoiTaoDonEmail,
			&order.NgayTao,
		); err != nil {
			return 0, fmt.Errorf("error scanning pending order before placing: %w", err)
		}
		selectedOrders = append(selectedOrders, order)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating pending orders before placing: %w", err)
	}

	if len(selectedOrders) == 0 {
		return 0, fmt.Errorf("no pending orders found")
	}

	placedAt := currentTimestamp()
	for _, order := range selectedOrders {
		if _, err := tx.Exec(`
			INSERT INTO order_history (
				pending_order_id,
				company_contact_id,
				nha_thau,
				ma_quan_ly,
				ma_vtyt_cu,
				ten_vtyt_bv,
				ma_hieu,
				hang_sx,
				don_vi_tinh,
				quy_cach,
				so_luong,
				email,
				source,
				nguoi_phe_duyet_id,
				nguoi_phe_duyet,
				nguoi_phe_duyet_email,
				thoi_gian_phe_duyet,
				nguoi_tao_don_id,
				nguoi_tao_don,
				nguoi_tao_don_email,
				ngay_tao,
				ngay_dat_hang,
				trang_thai,
				email_sent,
				nguoi_dat_hang_id,
				nguoi_dat_hang,
				nguoi_dat_hang_email
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			order.ID,
			nullInt64ToValue(order.CompanyContactID),
			order.NhaThau,
			order.MaQuanLy,
			order.MaVtytCu,
			order.TenVtytBv,
			order.MaHieu,
			order.HangSx,
			order.DonViTinh,
			order.QuyCach,
			order.DotGoiHang,
			order.Email,
			order.Source,
			nullInt64ToValue(order.NguoiPheDuyetID),
			order.NguoiPheDuyet,
			order.NguoiPheDuyetEmail,
			order.ThoiGianPheDuyet,
			nullInt64ToValue(order.NguoiTaoDonID),
			order.NguoiTaoDon,
			order.NguoiTaoDonEmail,
			order.NgayTao,
			placedAt,
			"Đã gửi email",
			1,
			placedBy.ID,
			placedBy.Username,
			placedBy.Email,
		); err != nil {
			return 0, fmt.Errorf("error inserting order history: %w", err)
		}
	}

	deleteQuery := fmt.Sprintf(`DELETE FROM pending_orders WHERE id IN (%s)`, placeholders)
	if _, err := tx.Exec(deleteQuery, args...); err != nil {
		return 0, fmt.Errorf("error deleting placed pending orders: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing placed orders: %w", err)
	}

	return len(selectedOrders), nil
}

func (r *OrderRepository) insertPendingOrderTx(tx *sql.Tx, input CreatePendingOrderInput, now string) error {
	approverID := nullableInt64(input.Approver)
	approverName := nullableActorField(input.Approver, func(actor *OrderActor) string { return actor.Username })
	approverEmail := nullableActorField(input.Approver, func(actor *OrderActor) string { return actor.Email })
	contactRepo := NewCompanyContactRepository(r.DB)
	companyContactID, resolvedEmail, err := contactRepo.EnsureContactTx(tx, input.NhaThau, "", input.Email)
	if err != nil {
		return fmt.Errorf("error resolving company contact for pending order: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO pending_orders (
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ma_hieu,
			hang_sx,
			don_vi_tinh,
			quy_cach,
			so_luong,
			email,
			source,
			nguoi_phe_duyet_id,
			nguoi_phe_duyet,
			nguoi_phe_duyet_email,
			thoi_gian_phe_duyet,
			nguoi_tao_don_id,
			nguoi_tao_don,
			nguoi_tao_don_email,
			ngay_tao,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nullInt64ToValue(companyContactID),
		input.NhaThau,
		input.MaQuanLy,
		input.MaVtytCu,
		input.TenVtytBv,
		input.MaHieu,
		input.HangSx,
		input.DonViTinh,
		input.QuyCach,
		input.DotGoiHang,
		resolvedEmail,
		input.Source,
		approverID,
		approverName,
		approverEmail,
		input.ApprovalTime,
		input.CreatedBy.ID,
		input.CreatedBy.Username,
		input.CreatedBy.Email,
		now,
		now,
	); err != nil {
		return fmt.Errorf("error inserting pending order: %w", err)
	}

	return nil
}

func (r *OrderRepository) ensureQuantityColumn(tableName string) error {
	exists, err := r.tableExists(tableName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	hasQuantityColumn, err := r.columnExists(tableName, "so_luong")
	if err != nil {
		return err
	}
	if hasQuantityColumn {
		return nil
	}

	hasLegacyColumn, err := r.columnExists(tableName, "dot_goi_hang")
	if err != nil {
		return err
	}

	var statement string
	if hasLegacyColumn {
		statement = fmt.Sprintf(
			"ALTER TABLE %s CHANGE COLUMN dot_goi_hang so_luong INT NOT NULL",
			tableName,
		)
	} else {
		statement = fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN so_luong INT NOT NULL DEFAULT 0 AFTER quy_cach",
			tableName,
		)
	}

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring %s.so_luong: %w", tableName, err)
	}

	return nil
}

func (r *OrderRepository) tableExists(tableName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?
	`, tableName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking table %s: %w", tableName, err)
	}

	return count > 0, nil
}

func (r *OrderRepository) columnExists(tableName, columnName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, tableName, columnName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking column %s.%s: %w", tableName, columnName, err)
	}

	return count > 0, nil
}

func currentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

func nullableInt64(actor *OrderActor) interface{} {
	if actor == nil || actor.ID == 0 {
		return nil
	}
	return actor.ID
}

func nullableActorField(actor *OrderActor, field func(*OrderActor) string) string {
	if actor == nil {
		return ""
	}
	return field(actor)
}

func nullInt64ToValue(value sql.NullInt64) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func makePlaceholders(count int) string {
	if count <= 0 {
		return ""
	}

	parts := make([]string, count)
	for index := range parts {
		parts[index] = "?"
	}
	return strings.Join(parts, ", ")
}
