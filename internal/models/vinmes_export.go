package models

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultVinmesKhoHang   = "Kho vật tư tiêu hao"
	defaultVinmesNguon     = "Mua"
	defaultVinmesLoaiPhieu = "Thanh toán"
	defaultVinmesKyHieu    = "không hiểu"
	defaultVinmesGoiThau   = "không thấy"
)

var tenderCodePattern = regexp.MustCompile(`\b(9528|9530|9532|9534)\b`)

type VinmesExportFilter struct {
	Month        int
	Year         int
	All          bool
	MaterialCode string
	Limit        int
}

type VinmesExportSource struct {
	ReconciliationID int64
	OrderHistoryID   int64
	OrderBatchKey    string
	InvoiceRowID     *int64
	InvoiceIDHoaDon  string
	InvoiceNumber    string
	InvoiceKyHieu    string
	InvoiceDate      *time.Time
	InvoiceItemCode  string
	InvoiceItemName  string
	InvoiceQty       float64
	InvoiceTaxRate   *float64
	SupplierName     string
	MatchedAt        time.Time
}

type VinmesExportItem struct {
	GoiThau          string  `json:"goiThau"`
	KhoHang          string  `json:"khoHang"`
	Nguon            string  `json:"nguon"`
	NhaCungCap       string  `json:"nhaCungCap"`
	LoaiPhieu        string  `json:"loaiPhieu"`
	SoPhieu          string  `json:"soPhieu"`
	NgayYeuCau       string  `json:"ngayYeuCau"`
	KyHieu           string  `json:"kyHieu"`
	SoHoaDon         string  `json:"soHoaDon"`
	NgayHoaDon       string  `json:"ngayHoaDon"`
	Thue             string  `json:"thue"`
	MaHang           string  `json:"maHang"`
	TenHangHoa       string  `json:"tenHangHoa"`
	SoLuong          float64 `json:"soLuong"`
	ReconciliationID int64   `json:"reconciliationId"`
	OrderHistoryID   int64   `json:"orderHistoryId"`
}

func (r *InvoiceReconciliationRepository) ListVinmesExportSources(filter VinmesExportFilter) ([]VinmesExportSource, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			r.id,
			r.order_history_id,
			r.order_batch_key,
			r.invoice_row_id,
			r.invoice_id_hoa_don,
			COALESCE(NULLIF(r.invoice_number, ''), NULLIF(h_row.so_hoa_don, ''), NULLIF(h_invoice.so_hoa_don, '')) AS invoice_number,
			COALESCE(NULLIF(h_row.kyhieu, ''), NULLIF(h_invoice.kyhieu, ''), '') AS invoice_ky_hieu,
			COALESCE(h_row.ngay_hoa_don, h_invoice.ngay_hoa_don, r.invoice_time) AS invoice_date,
			COALESCE(NULLIF(r.invoice_item_code, ''), NULLIF(h_row.ma_hang_hoa, '')) AS invoice_item_code,
			COALESCE(NULLIF(r.invoice_item_name, ''), NULLIF(h_row.ten_hang_hoa, '')) AS invoice_item_name,
			CASE
				WHEN r.invoice_qty > 0 THEN r.invoice_qty
				ELSE COALESCE(h_row.so_luong, 0)
			END AS invoice_qty,
			h_row.thue_suat_gtgt,
			COALESCE(NULLIF(r.invoice_company_name, ''), NULLIF(h_row.cong_ty, ''), NULLIF(h_invoice.cong_ty, ''), NULLIF(r.nha_thau, '')) AS supplier_name,
			r.matched_at
		FROM order_invoice_reconciliation r
		LEFT JOIN hoa_don h_row ON h_row.id = r.invoice_row_id
		LEFT JOIN (
			SELECT
				id_hoa_don,
				MAX(so_hoa_don) AS so_hoa_don,
				MAX(kyhieu) AS kyhieu,
				MAX(ngay_hoa_don) AS ngay_hoa_don,
				MAX(cong_ty) AS cong_ty
			FROM hoa_don
			GROUP BY id_hoa_don
		) h_invoice ON h_invoice.id_hoa_don = r.invoice_id_hoa_don
		WHERE r.has_invoice = 1
		  AND r.status IN (?, ?)
	`)

	args := []interface{}{InvoiceReconciliationStatusDone, invoiceReconciliationLegacyStatusDone}

	if !filter.All {
		queryBuilder.WriteString(`
		  AND MONTH(r.matched_at) = ?
		  AND YEAR(r.matched_at) = ?
		`)
		args = append(args, filter.Month, filter.Year)
	}

	materialCode := strings.ToLower(strings.TrimSpace(filter.MaterialCode))
	if materialCode != "" {
		queryBuilder.WriteString(`
		  AND (
			LOWER(TRIM(COALESCE(r.invoice_item_code, ''))) = ?
			OR LOWER(TRIM(COALESCE(h_row.ma_hang_hoa, ''))) = ?
			OR LOWER(TRIM(COALESCE(r.ma_vtyt_cu, ''))) = ?
		  )
		`)
		args = append(args, materialCode, materialCode, materialCode)
	}

	queryBuilder.WriteString(`
		ORDER BY r.matched_at DESC, r.id DESC
	`)

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	queryBuilder.WriteString(" LIMIT ?")
	args = append(args, limit)

	rows, err := r.DB.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error listing vinmes export sources: %w", err)
	}
	defer rows.Close()

	items := make([]VinmesExportSource, 0)
	for rows.Next() {
		var item VinmesExportSource
		var invoiceRowID sql.NullInt64
		var invoiceDate sql.NullTime
		var invoiceTaxRate sql.NullFloat64

		if err := rows.Scan(
			&item.ReconciliationID,
			&item.OrderHistoryID,
			&item.OrderBatchKey,
			&invoiceRowID,
			&item.InvoiceIDHoaDon,
			&item.InvoiceNumber,
			&item.InvoiceKyHieu,
			&invoiceDate,
			&item.InvoiceItemCode,
			&item.InvoiceItemName,
			&item.InvoiceQty,
			&invoiceTaxRate,
			&item.SupplierName,
			&item.MatchedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning vinmes export source: %w", err)
		}

		if invoiceRowID.Valid {
			value := invoiceRowID.Int64
			item.InvoiceRowID = &value
		}
		if invoiceDate.Valid {
			value := invoiceDate.Time
			item.InvoiceDate = &value
		}
		if invoiceTaxRate.Valid {
			value := invoiceTaxRate.Float64
			item.InvoiceTaxRate = &value
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vinmes export sources: %w", err)
	}

	return items, nil
}

func BuildVinmesExportItem(source VinmesExportSource) VinmesExportItem {
	invoiceDate := ""
	if source.InvoiceDate != nil {
		invoiceDate = formatVinmesDate(*source.InvoiceDate)
	}

	return VinmesExportItem{
		GoiThau:          ExtractTenderReference(source.InvoiceItemName),
		KhoHang:          defaultVinmesKhoHang,
		Nguon:            defaultVinmesNguon,
		NhaCungCap:       strings.TrimSpace(source.SupplierName),
		LoaiPhieu:        defaultVinmesLoaiPhieu,
		SoPhieu:          buildVinmesRequestNumber(source.ReconciliationID, source.MatchedAt),
		NgayYeuCau:       formatVinmesDate(source.MatchedAt),
		KyHieu:           fallbackVinmesKyHieu(source.InvoiceKyHieu),
		SoHoaDon:         strings.TrimSpace(source.InvoiceNumber),
		NgayHoaDon:       invoiceDate,
		Thue:             formatVinmesTaxRate(source.InvoiceTaxRate),
		MaHang:           strings.TrimSpace(source.InvoiceItemCode),
		TenHangHoa:       strings.TrimSpace(source.InvoiceItemName),
		SoLuong:          source.InvoiceQty,
		ReconciliationID: source.ReconciliationID,
		OrderHistoryID:   source.OrderHistoryID,
	}
}

func fallbackVinmesKyHieu(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultVinmesKyHieu
	}
	return trimmed
}

func ExtractTenderReference(input string) string {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return defaultVinmesGoiThau
	}

	matches := tenderCodePattern.FindStringSubmatch(normalized)
	if len(matches) >= 2 {
		return matches[1] + "/QĐ-BV"
	}

	return defaultVinmesGoiThau
}

func formatVinmesTaxRate(rate *float64) string {
	if rate == nil {
		return "0%"
	}

	if *rate == float64(int64(*rate)) {
		return fmt.Sprintf("%d%%", int64(*rate))
	}

	return strconv.FormatFloat(*rate, 'f', -1, 64) + "%"
}

func formatVinmesDate(value time.Time) string {
	return value.Format("02/01/2006")
}

func buildVinmesRequestNumber(reconciliationID int64, matchedAt time.Time) string {
	return fmt.Sprintf("PN%s%06d", matchedAt.Format("20060102"), reconciliationID)
}
