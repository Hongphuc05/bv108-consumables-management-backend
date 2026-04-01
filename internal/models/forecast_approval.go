package models

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ForecastApprovalStatusApproved = "approved"
	ForecastApprovalStatusRejected = "rejected"
	ForecastApprovalStatusEdited   = "edited"
)

type ForecastApprovalRecord struct {
	ID              int64  `json:"id"`
	ForecastMonth   int    `json:"forecastMonth"`
	ForecastYear    int    `json:"forecastYear"`
	MaQuanLy        string `json:"maQuanLy"`
	MaVtytCu        string `json:"maVtytCu"`
	TenVtytBv       string `json:"tenVtytBv"`
	Status          string `json:"status"`
	LyDo            string `json:"lyDo,omitempty"`
	DuTruGoc        *int   `json:"duTruGoc,omitempty"`
	DuTruSua        *int   `json:"duTruSua,omitempty"`
	NguoiDuyet      string `json:"nguoiDuyet"`
	NguoiDuyetEmail string `json:"nguoiDuyetEmail,omitempty"`
	ThoiGianDuyet   string `json:"thoiGianDuyet"`
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

type ForecastChangeHistoryRecord struct {
	ID                 int64  `json:"id"`
	ForecastMonth      int    `json:"forecastMonth"`
	ForecastYear       int    `json:"forecastYear"`
	MaQuanLy           string `json:"maQuanLy"`
	MaVtytCu           string `json:"maVtytCu"`
	TenVtytBv          string `json:"tenVtytBv"`
	ActionType         string `json:"actionType"`
	StatusBefore       string `json:"statusBefore,omitempty"`
	StatusAfter        string `json:"statusAfter"`
	DuTruGoc           *int   `json:"duTruGoc,omitempty"`
	DuTruSua           *int   `json:"duTruSua,omitempty"`
	NguoiThucHien      string `json:"nguoiThucHien"`
	NguoiThucHienEmail string `json:"nguoiThucHienEmail,omitempty"`
	ThoiGianThucHien   string `json:"thoiGianThucHien"`
}

type ForecastMonthlyHistoryItem struct {
	STT        int64  `json:"stt"`
	MaVtyt     string `json:"maVtyt"`
	TenVtyt    string `json:"tenVtyt"`
	QuyCach    string `json:"quyCach"`
	DonViTinh  string `json:"donViTinh"`
	DuTru      int    `json:"duTru"`
	GoiHang    int    `json:"goiHang"`
	DonGia     int64  `json:"donGia"`
	ThanhTien  int64  `json:"thanhTien"`
	TrangThai  string `json:"trangThai"`
	NguoiDuyet string `json:"nguoiDuyet"`
	NgayDuyet  string `json:"ngayDuyet"`
}

type ForecastMonthlyHistoryRecord struct {
	ID            string                       `json:"id"`
	Thang         int                          `json:"thang"`
	Nam           int                          `json:"nam"`
	NgayTao       string                       `json:"ngayTao"`
	NgayDuyet     string                       `json:"ngayDuyet"`
	NguoiTao      string                       `json:"nguoiTao"`
	NguoiDuyet    string                       `json:"nguoiDuyet"`
	TongSoVatTu   int                          `json:"tongSoVatTu"`
	TongGiaTri    int64                        `json:"tongGiaTri"`
	TrangThai     string                       `json:"trangThai"`
	DanhSachVatTu []ForecastMonthlyHistoryItem `json:"danhSachVatTu"`
}

type ForecastApprovalRepository struct {
	DB *sql.DB
}

func NewForecastApprovalRepository(db *sql.DB) *ForecastApprovalRepository {
	return &ForecastApprovalRepository{DB: db}
}

func (r *ForecastApprovalRepository) EnsureSchema() error {
	approvalsStatement := `
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

	if _, err := r.DB.Exec(approvalsStatement); err != nil {
		return fmt.Errorf("error ensuring forecast approvals schema: %w", err)
	}

	historyStatement := `
		CREATE TABLE IF NOT EXISTS forecast_change_history (
			id BIGINT NOT NULL AUTO_INCREMENT,
			forecast_year INT NOT NULL,
			forecast_month INT NOT NULL,
			ma_quan_ly VARCHAR(255) NOT NULL DEFAULT '',
			ma_vtyt_cu VARCHAR(255) NOT NULL,
			ten_vtyt_bv VARCHAR(500) NOT NULL,
			du_tru_goc INT NULL,
			du_tru_sua INT NULL,
			nguoi_thuc_hien_id BIGINT NOT NULL,
			nguoi_thuc_hien VARCHAR(255) NOT NULL,
			nguoi_thuc_hien_email VARCHAR(255) NOT NULL DEFAULT '',
			thoi_gian_thuc_hien DATETIME NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_forecast_change_history_period (forecast_year, forecast_month),
			KEY idx_forecast_change_history_item (ma_vtyt_cu),
			KEY idx_forecast_change_history_lookup (forecast_year, forecast_month, ma_quan_ly, ma_vtyt_cu, thoi_gian_thuc_hien, id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(historyStatement); err != nil {
		return fmt.Errorf("error ensuring forecast change history schema: %w", err)
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

		if shouldPersistForecastChange(input) {
			if _, err := tx.Exec(`
				INSERT INTO forecast_change_history (
					forecast_year,
					forecast_month,
					ma_quan_ly,
					ma_vtyt_cu,
					ten_vtyt_bv,
					du_tru_goc,
					du_tru_sua,
					nguoi_thuc_hien_id,
					nguoi_thuc_hien,
					nguoi_thuc_hien_email,
					thoi_gian_thuc_hien
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				input.ForecastYear,
				input.ForecastMonth,
				input.MaQuanLy,
				input.MaVtytCu,
				input.TenVtytBv,
				nullableIntPointer(input.DuTruGoc),
				nullableIntPointer(input.DuTruSua),
				input.Reviewer.ID,
				input.Reviewer.Username,
				input.Reviewer.Email,
				time.Now(),
			); err != nil {
				return fmt.Errorf("error saving forecast change history: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing forecast approvals: %w", err)
	}

	return nil
}

func (r *ForecastApprovalRepository) ListChangeHistory(limit, month, year int, latestOnly bool) ([]ForecastChangeHistoryRecord, error) {
	if !latestOnly && (limit <= 0 || limit > 5000) {
		limit = 1000
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			id,
			forecast_month,
			forecast_year,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			du_tru_goc,
			du_tru_sua,
			nguoi_thuc_hien,
			nguoi_thuc_hien_email,
			DATE_FORMAT(thoi_gian_thuc_hien, '%Y-%m-%dT%H:%i:%s')
		FROM forecast_change_history
	`)

	whereClauses := make([]string, 0, 2)
	args := make([]interface{}, 0, 3)
	if month > 0 {
		whereClauses = append(whereClauses, "forecast_month = ?")
		args = append(args, month)
	}
	if year > 0 {
		whereClauses = append(whereClauses, "forecast_year = ?")
		args = append(args, year)
	}
	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY thoi_gian_thuc_hien DESC, id DESC")
	if !latestOnly {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, limit)
	}

	rows, err := r.DB.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error listing forecast change history: %w", err)
	}
	defer rows.Close()

	records := make([]ForecastChangeHistoryRecord, 0)
	seenLatestKeys := map[string]struct{}{}
	for rows.Next() {
		var row ForecastChangeHistoryRecord
		var duTruGoc sql.NullInt64
		var duTruSua sql.NullInt64
		if err := rows.Scan(
			&row.ID,
			&row.ForecastMonth,
			&row.ForecastYear,
			&row.MaQuanLy,
			&row.MaVtytCu,
			&row.TenVtytBv,
			&duTruGoc,
			&duTruSua,
			&row.NguoiThucHien,
			&row.NguoiThucHienEmail,
			&row.ThoiGianThucHien,
		); err != nil {
			return nil, fmt.Errorf("error scanning forecast change history: %w", err)
		}

		if duTruGoc.Valid {
			value := int(duTruGoc.Int64)
			row.DuTruGoc = &value
		}
		if duTruSua.Valid {
			value := int(duTruSua.Int64)
			row.DuTruSua = &value
		}

		row.ActionType = "edit"
		row.StatusAfter = ForecastApprovalStatusEdited

		if latestOnly {
			recordKey := forecastChangeHistoryKey(row.ForecastMonth, row.ForecastYear, row.MaQuanLy, row.MaVtytCu)
			if _, exists := seenLatestKeys[recordKey]; exists {
				continue
			}
			seenLatestKeys[recordKey] = struct{}{}
		}

		records = append(records, row)
		if latestOnly && limit > 0 && len(records) >= limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating forecast change history: %w", err)
	}

	if latestOnly {
		return records, nil
	}

	approvalRecords, err := r.listApprovalHistory(limit, month, year)
	if err != nil {
		return nil, err
	}

	records = append(records, approvalRecords...)
	sort.SliceStable(records, func(i, j int) bool {
		leftTime := parseForecastHistoryTime(records[i].ThoiGianThucHien)
		rightTime := parseForecastHistoryTime(records[j].ThoiGianThucHien)
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return records[i].ID > records[j].ID
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

func (r *ForecastApprovalRepository) listApprovalHistory(limit, month, year int) ([]ForecastChangeHistoryRecord, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			id,
			forecast_month,
			forecast_year,
			ma_quan_ly,
			ma_vtyt_cu,
			ten_vtyt_bv,
			du_tru_goc,
			du_tru_sua,
			nguoi_duyet,
			nguoi_duyet_email,
			thoi_gian_duyet
		FROM forecast_approvals
		WHERE status = ?
	`)

	args := []interface{}{ForecastApprovalStatusApproved}
	if month > 0 {
		queryBuilder.WriteString(" AND forecast_month = ?")
		args = append(args, month)
	}
	if year > 0 {
		queryBuilder.WriteString(" AND forecast_year = ?")
		args = append(args, year)
	}

	queryBuilder.WriteString(" ORDER BY updated_at DESC, id DESC")
	if limit > 0 {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, limit)
	}

	rows, err := r.DB.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error listing forecast approval history: %w", err)
	}
	defer rows.Close()

	records := make([]ForecastChangeHistoryRecord, 0)
	for rows.Next() {
		var row ForecastChangeHistoryRecord
		var rawID int64
		var duTruGoc sql.NullInt64
		var duTruSua sql.NullInt64
		if err := rows.Scan(
			&rawID,
			&row.ForecastMonth,
			&row.ForecastYear,
			&row.MaQuanLy,
			&row.MaVtytCu,
			&row.TenVtytBv,
			&duTruGoc,
			&duTruSua,
			&row.NguoiThucHien,
			&row.NguoiThucHienEmail,
			&row.ThoiGianThucHien,
		); err != nil {
			return nil, fmt.Errorf("error scanning forecast approval history: %w", err)
		}

		row.ID = -rawID
		row.ActionType = "approve"
		row.StatusAfter = ForecastApprovalStatusApproved

		if duTruGoc.Valid {
			value := int(duTruGoc.Int64)
			row.DuTruGoc = &value
		}
		if duTruSua.Valid {
			value := int(duTruSua.Int64)
			row.DuTruSua = &value
		}

		records = append(records, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating forecast approval history: %w", err)
	}

	return records, nil
}

func (r *ForecastApprovalRepository) ListMonthlyChangeHistory() ([]ForecastMonthlyHistoryRecord, error) {
	rows, err := r.DB.Query(`
		SELECT
			fa.id,
			fa.forecast_month,
			fa.forecast_year,
			fa.ma_vtyt_cu,
			fa.ten_vtyt_bv,
			fa.status,
			COALESCE(h.du_tru_goc, fa.du_tru_goc),
			COALESCE(h.du_tru_sua, fa.du_tru_sua),
			fa.nguoi_duyet,
			fa.thoi_gian_duyet,
			COALESCE(s.QUY_CACH_DONG_GOI, ''),
			COALESCE(s.UNIT, ''),
			COALESCE(s.PRICE, 0)
		FROM forecast_approvals fa
		LEFT JOIN (
			SELECT
				h1.forecast_year,
				h1.forecast_month,
				h1.ma_vtyt_cu,
				h1.du_tru_goc,
				h1.du_tru_sua
			FROM forecast_change_history h1
			INNER JOIN (
				SELECT MAX(id) AS latest_id
				FROM forecast_change_history
				GROUP BY forecast_year, forecast_month, ma_vtyt_cu
			) latest ON latest.latest_id = h1.id
		) h ON h.forecast_year = fa.forecast_year
			AND h.forecast_month = fa.forecast_month
			AND TRIM(COALESCE(h.ma_vtyt_cu, '')) = TRIM(COALESCE(fa.ma_vtyt_cu, ''))
		LEFT JOIN supplies s ON TRIM(COALESCE(s.ID, '')) = TRIM(fa.ma_vtyt_cu)
		ORDER BY fa.forecast_year DESC, fa.forecast_month DESC, fa.updated_at DESC, fa.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing monthly forecast change history: %w", err)
	}
	defer rows.Close()

	type monthBucket struct {
		record           ForecastMonthlyHistoryRecord
		approvedOrEdited int
		rejected         int
	}

	buckets := map[string]*monthBucket{}

	for rows.Next() {
		var (
			itemID     int64
			month      int
			year       int
			maVtyt     string
			tenVtyt    string
			status     string
			duTruGoc   sql.NullInt64
			duTruSua   sql.NullInt64
			nguoiDuyet string
			ngayDuyet  string
			quyCach    string
			donViTinh  string
			donGia     float64
		)

		if err := rows.Scan(
			&itemID,
			&month,
			&year,
			&maVtyt,
			&tenVtyt,
			&status,
			&duTruGoc,
			&duTruSua,
			&nguoiDuyet,
			&ngayDuyet,
			&quyCach,
			&donViTinh,
			&donGia,
		); err != nil {
			return nil, fmt.Errorf("error scanning monthly forecast change history row: %w", err)
		}

		key := fmt.Sprintf("forecast-%d-%02d", year, month)
		bucket, exists := buckets[key]
		if !exists {
			bucket = &monthBucket{
				record: ForecastMonthlyHistoryRecord{
					ID:            key,
					Thang:         month,
					Nam:           year,
					NgayTao:       fmt.Sprintf("%04d-%02d-01T00:00:00Z", year, month),
					NgayDuyet:     ngayDuyet,
					NguoiTao:      "Hệ thống",
					NguoiDuyet:    nguoiDuyet,
					TongSoVatTu:   0,
					TongGiaTri:    0,
					TrangThai:     "partial",
					DanhSachVatTu: []ForecastMonthlyHistoryItem{},
				},
			}
			buckets[key] = bucket
		}

		duTru := 0
		if duTruSua.Valid {
			duTru = int(duTruSua.Int64)
		} else if duTruGoc.Valid {
			duTru = int(duTruGoc.Int64)
		}

		goiHang := duTru
		unitPrice := int64(math.Round(donGia))
		thanhTien := int64(goiHang) * unitPrice

		bucket.record.DanhSachVatTu = append(bucket.record.DanhSachVatTu, ForecastMonthlyHistoryItem{
			STT:        itemID,
			MaVtyt:     maVtyt,
			TenVtyt:    tenVtyt,
			QuyCach:    quyCach,
			DonViTinh:  donViTinh,
			DuTru:      duTru,
			GoiHang:    goiHang,
			DonGia:     unitPrice,
			ThanhTien:  thanhTien,
			TrangThai:  status,
			NguoiDuyet: nguoiDuyet,
			NgayDuyet:  ngayDuyet,
		})

		bucket.record.TongSoVatTu++
		bucket.record.TongGiaTri += thanhTien
		if bucket.record.NguoiDuyet == "" {
			bucket.record.NguoiDuyet = nguoiDuyet
		}
		if bucket.record.NgayDuyet == "" || bucket.record.NgayDuyet < ngayDuyet {
			bucket.record.NgayDuyet = ngayDuyet
		}

		if status == ForecastApprovalStatusRejected {
			bucket.rejected++
		} else {
			bucket.approvedOrEdited++
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly forecast change history rows: %w", err)
	}

	records := make([]ForecastMonthlyHistoryRecord, 0, len(buckets))
	for _, bucket := range buckets {
		total := bucket.record.TongSoVatTu
		if total > 0 {
			if bucket.rejected == total {
				bucket.record.TrangThai = "rejected"
			} else if bucket.approvedOrEdited == total {
				bucket.record.TrangThai = "approved"
			} else {
				bucket.record.TrangThai = "partial"
			}
		}

		records = append(records, bucket.record)
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Nam != records[j].Nam {
			return records[i].Nam > records[j].Nam
		}
		if records[i].Thang != records[j].Thang {
			return records[i].Thang > records[j].Thang
		}
		return records[i].ID > records[j].ID
	})

	return records, nil
}

func actionTypeFromStatus(status string) string {
	switch status {
	case ForecastApprovalStatusApproved:
		return "approve"
	case ForecastApprovalStatusRejected:
		return "reject"
	case ForecastApprovalStatusEdited:
		return "edit"
	default:
		return "approve"
	}
}

func shouldPersistForecastChange(input SaveForecastApprovalInput) bool {
	return input.Status == ForecastApprovalStatusEdited || input.DuTruGoc != nil || input.DuTruSua != nil
}

func forecastChangeHistoryKey(month, year int, maQuanLy, maVtytCu string) string {
	return fmt.Sprintf("%d-%02d-%s-%s", year, month, strings.TrimSpace(maQuanLy), strings.TrimSpace(maVtytCu))
}

func parseForecastHistoryTime(value string) time.Time {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}

	return time.Time{}
}

var packSizeRegex = regexp.MustCompile(`\d+`)

func extractPackQuantity(value string) int {
	match := packSizeRegex.FindString(value)
	if match == "" {
		return 1
	}

	parsed, err := strconv.Atoi(match)
	if err != nil || parsed <= 0 {
		return 1
	}

	return parsed
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
