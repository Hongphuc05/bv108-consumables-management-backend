package models

import (
	"database/sql"
	"fmt"
	"time"
)

type InvoiceReconciliationRecord struct {
	ID                      int64      `json:"id"`
	OrderHistoryID          int64      `json:"orderHistoryId"`
	OrderBatchKey           string     `json:"orderBatchKey"`
	CompanyContactID        *int64     `json:"companyContactId,omitempty"`
	NhaThau                 string     `json:"nhaThau"`
	MaQuanLy                string     `json:"maQuanLy"`
	MaVtytCu                string     `json:"maVtytCu"`
	TenVtytBv               string     `json:"tenVtytBv"`
	OrderedQty              int        `json:"orderedQty"`
	OrderTime               *time.Time `json:"orderTime,omitempty"`
	InvoiceNumber           string     `json:"invoiceNumber"`
	InvoiceIDHoaDon         string     `json:"invoiceIdHoaDon,omitempty"`
	InvoiceRowID            *int64     `json:"invoiceRowId,omitempty"`
	InvoiceCompanyContactID *int64     `json:"invoiceCompanyContactId,omitempty"`
	InvoiceCompanyName      string     `json:"invoiceCompanyName,omitempty"`
	InvoiceItemCode         string     `json:"invoiceItemCode,omitempty"`
	InvoiceItemName         string     `json:"invoiceItemName,omitempty"`
	InvoiceQty              float64    `json:"invoiceQty"`
	InvoiceTime             *time.Time `json:"invoiceTime,omitempty"`
	HasInvoice              bool       `json:"hasInvoice"`
	DetailStatus            string     `json:"detailStatus"`
	DetailNote              string     `json:"detailNote,omitempty"`
	MatchScore              float64    `json:"matchScore"`
	QuantityDiff            float64    `json:"quantityDiff"`
	MatchedByUserID         *int64     `json:"matchedByUserId,omitempty"`
	MatchedByUsername       string     `json:"matchedByUsername"`
	MatchedByEmail          string     `json:"matchedByEmail,omitempty"`
	MatchedAt               time.Time  `json:"matchedAt"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

type UpsertInvoiceReconciliationInput struct {
	OrderHistoryID          int64
	OrderBatchKey           string
	CompanyContactID        *int64
	NhaThau                 string
	MaQuanLy                string
	MaVtytCu                string
	TenVtytBv               string
	OrderedQty              int
	OrderTime               *time.Time
	InvoiceNumber           string
	InvoiceIDHoaDon         string
	InvoiceRowID            *int64
	InvoiceCompanyContactID *int64
	InvoiceCompanyName      string
	InvoiceItemCode         string
	InvoiceItemName         string
	InvoiceQty              float64
	InvoiceTime             *time.Time
	HasInvoice              bool
	DetailStatus            string
	DetailNote              string
	MatchScore              float64
	QuantityDiff            float64
	MatchedByUserID         *int64
	MatchedByUsername       string
	MatchedByEmail          string
	MatchedAt               time.Time
}

type InvoiceReconciliationRepository struct {
	DB *sql.DB
}

func NewInvoiceReconciliationRepository(db *sql.DB) *InvoiceReconciliationRepository {
	return &InvoiceReconciliationRepository{DB: db}
}

func (r *InvoiceReconciliationRepository) EnsureSchema() error {
	statement := `
		CREATE TABLE IF NOT EXISTS order_invoice_reconciliation (
			id BIGINT NOT NULL AUTO_INCREMENT,
			order_history_id BIGINT NOT NULL,
			order_batch_key VARCHAR(255) NOT NULL DEFAULT '',
			company_contact_id BIGINT NULL,
			nha_thau VARCHAR(255) NOT NULL,
			ma_quan_ly VARCHAR(255) NOT NULL DEFAULT '',
			ma_vtyt_cu VARCHAR(255) NOT NULL,
			ten_vtyt_bv VARCHAR(500) NOT NULL,
			ordered_qty INT NOT NULL,
			order_time DATETIME NULL,
			invoice_number VARCHAR(128) NOT NULL,
			invoice_id_hoa_don VARCHAR(128) NOT NULL DEFAULT '',
			invoice_row_id BIGINT NULL,
			invoice_company_contact_id BIGINT NULL,
			invoice_company_name VARCHAR(255) NOT NULL DEFAULT '',
			invoice_item_code VARCHAR(255) NOT NULL DEFAULT '',
			invoice_item_name VARCHAR(500) NOT NULL DEFAULT '',
			invoice_qty DECIMAL(18,3) NOT NULL DEFAULT 0,
			invoice_time DATETIME NULL,
			has_invoice TINYINT(1) NOT NULL DEFAULT 0,
			detail_status VARCHAR(64) NOT NULL DEFAULT '',
			detail_note VARCHAR(500) NOT NULL DEFAULT '',
			match_score DECIMAL(10,2) NOT NULL DEFAULT 0,
			quantity_diff DECIMAL(18,3) NOT NULL DEFAULT 0,
			matched_by_user_id BIGINT NULL,
			matched_by_username VARCHAR(255) NOT NULL DEFAULT '',
			matched_by_email VARCHAR(255) NOT NULL DEFAULT '',
			matched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uq_order_invoice_match (order_history_id, order_batch_key, invoice_number),
			KEY idx_oir_matched_at (matched_at),
			KEY idx_oir_invoice_time (invoice_time),
			KEY idx_oir_status (detail_status),
			KEY idx_oir_company_contact (company_contact_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring order_invoice_reconciliation schema: %w", err)
	}

	return nil
}

func (r *InvoiceReconciliationRepository) UpsertBulk(inputs []UpsertInvoiceReconciliationInput) error {
	if len(inputs) == 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting invoice reconciliation transaction: %w", err)
	}
	defer tx.Rollback()

	for _, item := range inputs {
		if _, err := tx.Exec(`
			INSERT INTO order_invoice_reconciliation (
				order_history_id,
				order_batch_key,
				company_contact_id,
				nha_thau,
				ma_quan_ly,
				ma_vtyt_cu,
				ten_vtyt_bv,
				ordered_qty,
				order_time,
				invoice_number,
				invoice_id_hoa_don,
				invoice_row_id,
				invoice_company_contact_id,
				invoice_company_name,
				invoice_item_code,
				invoice_item_name,
				invoice_qty,
				invoice_time,
				has_invoice,
				detail_status,
				detail_note,
				match_score,
				quantity_diff,
				matched_by_user_id,
				matched_by_username,
				matched_by_email,
				matched_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				company_contact_id = VALUES(company_contact_id),
				nha_thau = VALUES(nha_thau),
				ma_quan_ly = VALUES(ma_quan_ly),
				ma_vtyt_cu = VALUES(ma_vtyt_cu),
				ten_vtyt_bv = VALUES(ten_vtyt_bv),
				ordered_qty = VALUES(ordered_qty),
				order_time = VALUES(order_time),
				invoice_id_hoa_don = VALUES(invoice_id_hoa_don),
				invoice_row_id = VALUES(invoice_row_id),
				invoice_company_contact_id = VALUES(invoice_company_contact_id),
				invoice_company_name = VALUES(invoice_company_name),
				invoice_item_code = VALUES(invoice_item_code),
				invoice_item_name = VALUES(invoice_item_name),
				invoice_qty = VALUES(invoice_qty),
				invoice_time = VALUES(invoice_time),
				has_invoice = VALUES(has_invoice),
				detail_status = VALUES(detail_status),
				detail_note = VALUES(detail_note),
				match_score = VALUES(match_score),
				quantity_diff = VALUES(quantity_diff),
				matched_by_user_id = VALUES(matched_by_user_id),
				matched_by_username = VALUES(matched_by_username),
				matched_by_email = VALUES(matched_by_email),
				matched_at = VALUES(matched_at)
		`,
			item.OrderHistoryID,
			item.OrderBatchKey,
			nullableInt64Value(item.CompanyContactID),
			item.NhaThau,
			item.MaQuanLy,
			item.MaVtytCu,
			item.TenVtytBv,
			item.OrderedQty,
			nullableTimeValue(item.OrderTime),
			item.InvoiceNumber,
			item.InvoiceIDHoaDon,
			nullableInt64Value(item.InvoiceRowID),
			nullableInt64Value(item.InvoiceCompanyContactID),
			item.InvoiceCompanyName,
			item.InvoiceItemCode,
			item.InvoiceItemName,
			item.InvoiceQty,
			nullableTimeValue(item.InvoiceTime),
			boolToTinyInt(item.HasInvoice),
			item.DetailStatus,
			item.DetailNote,
			item.MatchScore,
			item.QuantityDiff,
			nullableInt64Value(item.MatchedByUserID),
			item.MatchedByUsername,
			item.MatchedByEmail,
			item.MatchedAt,
		); err != nil {
			return fmt.Errorf("error upserting invoice reconciliation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing invoice reconciliation transaction: %w", err)
	}

	return nil
}

func (r *InvoiceReconciliationRepository) ListByMonthYear(month, year int) ([]InvoiceReconciliationRecord, error) {
	rows, err := r.DB.Query(`
		SELECT
			id,
			order_history_id,
			order_batch_key,
			company_contact_id,
			nha_thau,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			ordered_qty,
			order_time,
			invoice_number,
			invoice_id_hoa_don,
			invoice_row_id,
			invoice_company_contact_id,
			invoice_company_name,
			invoice_item_code,
			invoice_item_name,
			invoice_qty,
			invoice_time,
			has_invoice,
			detail_status,
			detail_note,
			match_score,
			quantity_diff,
			matched_by_user_id,
			matched_by_username,
			matched_by_email,
			matched_at,
			created_at,
			updated_at
		FROM order_invoice_reconciliation
		WHERE MONTH(matched_at) = ? AND YEAR(matched_at) = ?
		ORDER BY matched_at DESC, id DESC
	`, month, year)
	if err != nil {
		return nil, fmt.Errorf("error listing invoice reconciliation history: %w", err)
	}
	defer rows.Close()

	records := make([]InvoiceReconciliationRecord, 0)
	for rows.Next() {
		var item InvoiceReconciliationRecord
		var companyContactID sql.NullInt64
		var orderTime sql.NullTime
		var invoiceRowID sql.NullInt64
		var invoiceCompanyContactID sql.NullInt64
		var invoiceTime sql.NullTime
		var hasInvoice int
		var matchedByUserID sql.NullInt64

		if err := rows.Scan(
			&item.ID,
			&item.OrderHistoryID,
			&item.OrderBatchKey,
			&companyContactID,
			&item.NhaThau,
			&item.MaQuanLy,
			&item.MaVtytCu,
			&item.TenVtytBv,
			&item.OrderedQty,
			&orderTime,
			&item.InvoiceNumber,
			&item.InvoiceIDHoaDon,
			&invoiceRowID,
			&invoiceCompanyContactID,
			&item.InvoiceCompanyName,
			&item.InvoiceItemCode,
			&item.InvoiceItemName,
			&item.InvoiceQty,
			&invoiceTime,
			&hasInvoice,
			&item.DetailStatus,
			&item.DetailNote,
			&item.MatchScore,
			&item.QuantityDiff,
			&matchedByUserID,
			&item.MatchedByUsername,
			&item.MatchedByEmail,
			&item.MatchedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning invoice reconciliation history: %w", err)
		}

		if companyContactID.Valid {
			value := companyContactID.Int64
			item.CompanyContactID = &value
		}
		if orderTime.Valid {
			value := orderTime.Time
			item.OrderTime = &value
		}
		if invoiceRowID.Valid {
			value := invoiceRowID.Int64
			item.InvoiceRowID = &value
		}
		if invoiceCompanyContactID.Valid {
			value := invoiceCompanyContactID.Int64
			item.InvoiceCompanyContactID = &value
		}
		if invoiceTime.Valid {
			value := invoiceTime.Time
			item.InvoiceTime = &value
		}
		item.HasInvoice = hasInvoice == 1
		if matchedByUserID.Valid {
			value := matchedByUserID.Int64
			item.MatchedByUserID = &value
		}

		records = append(records, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invoice reconciliation history: %w", err)
	}

	return records, nil
}

func (r *InvoiceReconciliationRepository) ListMatchedInvoiceNumbers(month, year int) ([]string, error) {
	rows, err := r.DB.Query(`
		SELECT DISTINCT invoice_number
		FROM order_invoice_reconciliation
		WHERE has_invoice = 1
			AND MONTH(matched_at) = ?
			AND YEAR(matched_at) = ?
			AND invoice_number <> ''
		ORDER BY invoice_number ASC
	`, month, year)
	if err != nil {
		return nil, fmt.Errorf("error listing matched invoice numbers: %w", err)
	}
	defer rows.Close()

	invoiceNumbers := make([]string, 0)
	for rows.Next() {
		var invoiceNumber string
		if err := rows.Scan(&invoiceNumber); err != nil {
			return nil, fmt.Errorf("error scanning matched invoice number: %w", err)
		}
		invoiceNumbers = append(invoiceNumbers, invoiceNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matched invoice numbers: %w", err)
	}

	return invoiceNumbers, nil
}

func (r *InvoiceReconciliationRepository) ListAllMatchedInvoiceNumbers() ([]string, error) {
	rows, err := r.DB.Query(`
		SELECT DISTINCT invoice_number
		FROM order_invoice_reconciliation
		WHERE has_invoice = 1
			AND invoice_number <> ''
		ORDER BY invoice_number ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing all matched invoice numbers: %w", err)
	}
	defer rows.Close()

	invoiceNumbers := make([]string, 0)
	for rows.Next() {
		var invoiceNumber string
		if err := rows.Scan(&invoiceNumber); err != nil {
			return nil, fmt.Errorf("error scanning matched invoice number: %w", err)
		}
		invoiceNumbers = append(invoiceNumbers, invoiceNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matched invoice numbers: %w", err)
	}

	return invoiceNumbers, nil
}

func nullableInt64Value(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTimeValue(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func boolToTinyInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
