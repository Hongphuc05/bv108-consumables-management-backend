package models

import (
	"database/sql"
	"fmt"
)

const (
	ForecastApprovalStatusApproved = "approved"
	ForecastApprovalStatusRejected = "rejected"
	ForecastApprovalStatusEdited   = "edited"
)

type ForecastApprovalRecord struct {
	ID               int64  `json:"id"`
	ForecastMonth    int    `json:"forecastMonth"`
	ForecastYear     int    `json:"forecastYear"`
	MaQuanLy         string `json:"maQuanLy"`
	MaVtytCu         string `json:"maVtytCu"`
	TenVtytBv        string `json:"tenVtytBv"`
	Status           string `json:"status"`
	LyDo             string `json:"lyDo,omitempty"`
	DuTruGoc         *int   `json:"duTruGoc,omitempty"`
	DuTruSua         *int   `json:"duTruSua,omitempty"`
	NguoiDuyet       string `json:"nguoiDuyet"`
	NguoiDuyetEmail  string `json:"nguoiDuyetEmail,omitempty"`
	ThoiGianDuyet    string `json:"thoiGianDuyet"`
}

type SaveForecastApprovalInput struct {
	ForecastMonth int
	ForecastYear  int
	MaQuanLy      string
	MaVtytCu      string
	TenVtytBv     string
	Status        string
	LyDo          string
	DuTruGoc      *int
	DuTruSua      *int
	Reviewer      OrderActor
	ReviewedAt    string
}

type ForecastApprovalRepository struct {
	DB *sql.DB
}

func NewForecastApprovalRepository(db *sql.DB) *ForecastApprovalRepository {
	return &ForecastApprovalRepository{DB: db}
}

func (r *ForecastApprovalRepository) EnsureSchema() error {
	statement := `
		CREATE TABLE IF NOT EXISTS forecast_approvals (
			id BIGINT NOT NULL AUTO_INCREMENT,
			forecast_month INT NOT NULL,
			forecast_year INT NOT NULL,
			ma_quan_ly VARCHAR(255) NOT NULL DEFAULT '',
			ma_vtyt_cu VARCHAR(255) NOT NULL,
			ten_vtyt_bv VARCHAR(500) NOT NULL,
			status VARCHAR(32) NOT NULL,
			ly_do TEXT NULL,
			du_tru_goc INT NULL,
			du_tru_sua INT NULL,
			nguoi_duyet_id BIGINT NOT NULL,
			nguoi_duyet VARCHAR(255) NOT NULL,
			nguoi_duyet_email VARCHAR(255) NOT NULL DEFAULT '',
			thoi_gian_duyet VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_forecast_approvals_period_item (forecast_year, forecast_month, ma_vtyt_cu),
			KEY idx_forecast_approvals_period (forecast_year, forecast_month),
			KEY idx_forecast_approvals_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring forecast approvals schema: %w", err)
	}

	return nil
}

func (r *ForecastApprovalRepository) ListByMonthYear(month, year int) ([]ForecastApprovalRecord, error) {
	rows, err := r.DB.Query(`
		SELECT
			id,
			forecast_month,
			forecast_year,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			status,
			COALESCE(ly_do, ''),
			du_tru_goc,
			du_tru_sua,
			nguoi_duyet,
			nguoi_duyet_email,
			thoi_gian_duyet
		FROM forecast_approvals
		WHERE forecast_month = ? AND forecast_year = ?
		ORDER BY updated_at DESC, id DESC
	`, month, year)
	if err != nil {
		return nil, fmt.Errorf("error listing forecast approvals: %w", err)
	}
	defer rows.Close()

	records := make([]ForecastApprovalRecord, 0)
	for rows.Next() {
		var record ForecastApprovalRecord
		var duTruGoc sql.NullInt64
		var duTruSua sql.NullInt64
		if err := rows.Scan(
			&record.ID,
			&record.ForecastMonth,
			&record.ForecastYear,
			&record.MaQuanLy,
			&record.MaVtytCu,
			&record.TenVtytBv,
			&record.Status,
			&record.LyDo,
			&duTruGoc,
			&duTruSua,
			&record.NguoiDuyet,
			&record.NguoiDuyetEmail,
			&record.ThoiGianDuyet,
		); err != nil {
			return nil, fmt.Errorf("error scanning forecast approval: %w", err)
		}

		if duTruGoc.Valid {
			value := int(duTruGoc.Int64)
			record.DuTruGoc = &value
		}
		if duTruSua.Valid {
			value := int(duTruSua.Int64)
			record.DuTruSua = &value
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating forecast approvals: %w", err)
	}

	return records, nil
}

func (r *ForecastApprovalRepository) SaveApproval(input SaveForecastApprovalInput) error {
	return r.SaveApprovals([]SaveForecastApprovalInput{input})
}

func (r *ForecastApprovalRepository) SaveApprovals(inputs []SaveForecastApprovalInput) error {
	if len(inputs) == 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting forecast approval transaction: %w", err)
	}
	defer tx.Rollback()

	statement := `
		INSERT INTO forecast_approvals (
			forecast_month,
			forecast_year,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			status,
			ly_do,
			du_tru_goc,
			du_tru_sua,
			nguoi_duyet_id,
			nguoi_duyet,
			nguoi_duyet_email,
			thoi_gian_duyet
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			ma_quan_ly = VALUES(ma_quan_ly),
			ten_vtyt_bv = VALUES(ten_vtyt_bv),
			status = VALUES(status),
			ly_do = VALUES(ly_do),
			du_tru_goc = VALUES(du_tru_goc),
			du_tru_sua = VALUES(du_tru_sua),
			nguoi_duyet_id = VALUES(nguoi_duyet_id),
			nguoi_duyet = VALUES(nguoi_duyet),
			nguoi_duyet_email = VALUES(nguoi_duyet_email),
			thoi_gian_duyet = VALUES(thoi_gian_duyet)
	`

	for _, input := range inputs {
		if _, err := tx.Exec(
			statement,
			input.ForecastMonth,
			input.ForecastYear,
			input.MaQuanLy,
			input.MaVtytCu,
			input.TenVtytBv,
			input.Status,
			nullIfEmpty(input.LyDo),
			nullableIntPointer(input.DuTruGoc),
			nullableIntPointer(input.DuTruSua),
			input.Reviewer.ID,
			input.Reviewer.Username,
			input.Reviewer.Email,
			input.ReviewedAt,
		); err != nil {
			return fmt.Errorf("error saving forecast approval: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing forecast approvals: %w", err)
	}

	return nil
}

func nullableIntPointer(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
