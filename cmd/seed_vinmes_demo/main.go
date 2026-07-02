package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/models"
)

const demoSeedUsername = "vinmes_demo_seed"
const vinmesDemoCSVPathEnvKey = "VINMES_DEMO_CSV_PATH"

var demoMaterialCodes = []string{
	"D06041",
	"A00191",
	"A72781",
}

type csvInvoiceRow struct {
	TrangThaiHoaDon  string
	LoaiHoaDon       string
	SoHoaDon         string
	NgayHoaDon       time.Time
	MaSoThueNguoiBan string
	CongTy           string
	DiaChi           string
	LinkTraCuuHoaDon string
	IDHoaDon         string
	STTDongHang      int
	TenHangHoa       string
	MaHangHoa        string
	DonViTinh        string
	SoLuong          float64
	DonGiaChuaThue   float64
	ThueSuatGTGT     float64
}

type hoaDonRow struct {
	ID int64
	csvInvoiceRow
}

type seededOrder struct {
	OrderHistoryID int64
	OrderBatchKey  string
	HoaDon         hoaDonRow
}

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	userRepo := models.NewUserRepository(database.DB)
	orderRepo := models.NewOrderRepository(database.DB)
	invoiceRepo := models.NewInvoiceReconciliationRepository(database.DB)

	if err := orderRepo.EnsureSchema(); err != nil {
		log.Fatalf("ensure order schema: %v", err)
	}
	if err := invoiceRepo.EnsureSchema(); err != nil {
		log.Fatalf("ensure invoice reconciliation schema: %v", err)
	}

	actor, err := selectSeedActor(userRepo)
	if err != nil {
		log.Fatalf("select actor: %v", err)
	}

	csvPath, triedPaths, err := resolveDemoCSVPath()
	if err != nil {
		log.Fatalf("resolve csv path: %v (tried: %s)", err, strings.Join(triedPaths, "; "))
	}
	csvRows, err := loadCSVRows(csvPath, demoMaterialCodes)
	if err != nil {
		log.Fatalf("load csv rows: %v", err)
	}

	tx, err := database.DB.Begin()
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := cleanupExistingDemoData(tx); err != nil {
		log.Fatalf("cleanup demo data: %v", err)
	}

	now := time.Now()
	seededOrders := make([]seededOrder, 0, len(demoMaterialCodes))
	for _, code := range demoMaterialCodes {
		row, ok := csvRows[code]
		if !ok {
			log.Fatalf("missing csv row for material code %s", code)
		}

		hoaDonRecord, err := ensureHoaDonRow(tx, row)
		if err != nil {
			log.Fatalf("ensure hoa_don row %s: %v", code, err)
		}

		orderHistoryID, err := insertOrderHistory(tx, actor, hoaDonRecord, now)
		if err != nil {
			log.Fatalf("insert order history %s: %v", code, err)
		}

		seededOrders = append(seededOrders, seededOrder{
			OrderHistoryID: orderHistoryID,
			OrderBatchKey:  fmt.Sprintf("vinmes-demo-%s", strings.ToLower(code)),
			HoaDon:         hoaDonRecord,
		})
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit tx: %v", err)
	}

	inputs := make([]models.UpsertInvoiceReconciliationInput, 0, len(seededOrders))
	for _, item := range seededOrders {
		qtyInt := int(math.Round(item.HoaDon.SoLuong))
		inputs = append(inputs, models.UpsertInvoiceReconciliationInput{
			OrderHistoryID:          item.OrderHistoryID,
			OrderBatchKey:           item.OrderBatchKey,
			CompanyContactID:        nil,
			NhaThau:                 item.HoaDon.CongTy,
			MaQuanLy:                "",
			MaVtytCu:                item.HoaDon.MaHangHoa,
			TenVtytBv:               item.HoaDon.TenHangHoa,
			OrderedQty:              qtyInt,
			OrderTime:               &now,
			InvoiceNumber:           item.HoaDon.SoHoaDon,
			InvoiceIDHoaDon:         item.HoaDon.IDHoaDon,
			InvoiceRowID:            &item.HoaDon.ID,
			InvoiceCompanyContactID: nil,
			InvoiceCompanyName:      item.HoaDon.CongTy,
			InvoiceItemCode:         item.HoaDon.MaHangHoa,
			InvoiceItemName:         item.HoaDon.TenHangHoa,
			InvoiceQty:              item.HoaDon.SoLuong,
			InvoiceTime:             &item.HoaDon.NgayHoaDon,
			HasInvoice:              true,
			DetailStatus:            "matched",
			DetailNote:              "seeded from invoices_export.csv",
			MatchScore:              100,
			QuantityDiff:            0,
			MatchedByUserID:         &actor.ID,
			MatchedByUsername:       actor.Username,
			MatchedByEmail:          actor.Email,
			MatchedAt:               now,
			Note:                    fmt.Sprintf("demo export-to-vinmes %s", item.HoaDon.MaHangHoa),
			Status:                  models.InvoiceReconciliationStatusDone,
		})
	}

	if err := invoiceRepo.UpsertBulk(inputs); err != nil {
		log.Fatalf("upsert invoice reconciliations: %v", err)
	}

	fmt.Printf("Seeded %d Vinmes demo reconciliations for %04d-%02d\n", len(inputs), now.Year(), now.Month())
	for _, item := range seededOrders {
		fmt.Printf("- %s | invoice=%s | supplier=%s | order_history_id=%d\n", item.HoaDon.MaHangHoa, item.HoaDon.SoHoaDon, item.HoaDon.CongTy, item.OrderHistoryID)
	}
}

func selectSeedActor(userRepo *models.UserRepository) (*models.UserProfile, error) {
	users, err := userRepo.ListActiveUsers()
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no active users found")
	}

	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Role), "thu_kho") {
			userCopy := user
			return &userCopy, nil
		}
	}

	user := users[0]
	return &user, nil
}

func resolveDemoCSVPath() (string, []string, error) {
	candidates := make([]string, 0, 6)

	if envPath := strings.TrimSpace(os.Getenv(vinmesDemoCSVPathEnvKey)); envPath != "" {
		candidates = append(candidates, envPath)
	}

	candidates = append(candidates, filepath.Join("ubot-api", "invoices_export.csv"))

	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		candidates = append(candidates,
			filepath.Join(executableDir, "ubot-api", "invoices_export.csv"),
			filepath.Join(executableDir, "..", "ubot-api", "invoices_export.csv"),
		)
	}

	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		backendRoot := filepath.Join(filepath.Dir(sourceFile), "..", "..")
		candidates = append(candidates, filepath.Join(backendRoot, "ubot-api", "invoices_export.csv"))
	}

	tried := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}

		cleanPath := filepath.Clean(trimmed)
		if _, exists := seen[cleanPath]; exists {
			continue
		}
		seen[cleanPath] = struct{}{}
		tried = append(tried, cleanPath)

		info, err := os.Stat(cleanPath)
		if err == nil && !info.IsDir() {
			return cleanPath, tried, nil
		}
	}

	return "", tried, fmt.Errorf("could not locate invoices_export.csv")
}

func loadCSVRows(csvPath string, materialCodes []string) (map[string]csvInvoiceRow, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv has no data rows")
	}

	header := normalizeCSVHeader(rows[0])
	indexByName := make(map[string]int, len(header))
	for index, name := range header {
		indexByName[name] = index
	}

	needed := make(map[string]struct{}, len(materialCodes))
	for _, code := range materialCodes {
		needed[strings.TrimSpace(code)] = struct{}{}
	}

	result := make(map[string]csvInvoiceRow, len(materialCodes))
	for _, rawRow := range rows[1:] {
		code := strings.TrimSpace(csvValue(rawRow, indexByName, "ma hang hoa"))
		if code == "" {
			continue
		}
		if _, ok := needed[code]; !ok {
			continue
		}
		if _, exists := result[code]; exists {
			continue
		}

		ngayHoaDon, err := time.Parse(time.RFC3339, strings.TrimSpace(csvValue(rawRow, indexByName, "ngay hoa don")))
		if err != nil {
			return nil, fmt.Errorf("parse ngay hoa don for %s: %w", code, err)
		}

		sttDongHang, err := strconv.Atoi(strings.TrimSpace(csvValue(rawRow, indexByName, "stt dong hang")))
		if err != nil {
			return nil, fmt.Errorf("parse stt dong hang for %s: %w", code, err)
		}

		soLuong, err := strconv.ParseFloat(strings.TrimSpace(csvValue(rawRow, indexByName, "so luong")), 64)
		if err != nil {
			return nil, fmt.Errorf("parse so luong for %s: %w", code, err)
		}

		donGia, err := strconv.ParseFloat(strings.TrimSpace(csvValue(rawRow, indexByName, "don gia chua thue")), 64)
		if err != nil {
			return nil, fmt.Errorf("parse don gia for %s: %w", code, err)
		}

		thue, err := strconv.ParseFloat(strings.TrimSpace(csvValue(rawRow, indexByName, "thue suat gtgt")), 64)
		if err != nil {
			return nil, fmt.Errorf("parse thue for %s: %w", code, err)
		}

		result[code] = csvInvoiceRow{
			TrangThaiHoaDon:  strings.TrimSpace(csvValue(rawRow, indexByName, "trang thai hoa don")),
			LoaiHoaDon:       strings.TrimSpace(csvValue(rawRow, indexByName, "loai hoa don")),
			SoHoaDon:         strings.TrimSpace(csvValue(rawRow, indexByName, "so hoa don")),
			NgayHoaDon:       ngayHoaDon,
			MaSoThueNguoiBan: strings.TrimSpace(csvValue(rawRow, indexByName, "ma so thue nguoi ban")),
			CongTy:           strings.TrimSpace(csvValue(rawRow, indexByName, "cong ty")),
			DiaChi:           strings.TrimSpace(csvValue(rawRow, indexByName, "dia chi")),
			LinkTraCuuHoaDon: strings.TrimSpace(csvValue(rawRow, indexByName, "link tra cuu hoa don")),
			IDHoaDon:         strings.TrimSpace(csvValue(rawRow, indexByName, "id cua hoa don")),
			STTDongHang:      sttDongHang,
			TenHangHoa:       strings.TrimSpace(csvValue(rawRow, indexByName, "ten hang hoa")),
			MaHangHoa:        code,
			DonViTinh:        strings.TrimSpace(csvValue(rawRow, indexByName, "don vi tinh")),
			SoLuong:          soLuong,
			DonGiaChuaThue:   donGia,
			ThueSuatGTGT:     thue,
		}
	}

	for _, code := range materialCodes {
		if _, ok := result[code]; !ok {
			return nil, fmt.Errorf("material code %s not found in csv", code)
		}
	}

	return result, nil
}

func normalizeCSVHeader(header []string) []string {
	normalized := make([]string, len(header))
	for index, value := range header {
		value = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
		value = strings.ToLower(value)
		replacer := strings.NewReplacer(
			"á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a",
			"ă", "a", "ắ", "a", "ằ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
			"â", "a", "ấ", "a", "ầ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
			"é", "e", "è", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
			"ê", "e", "ế", "e", "ề", "e", "ể", "e", "ễ", "e", "ệ", "e",
			"í", "i", "ì", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
			"ó", "o", "ò", "o", "ỏ", "o", "õ", "o", "ọ", "o",
			"ô", "o", "ố", "o", "ồ", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
			"ơ", "o", "ớ", "o", "ờ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
			"ú", "u", "ù", "u", "ủ", "u", "ũ", "u", "ụ", "u",
			"ư", "u", "ứ", "u", "ừ", "u", "ử", "u", "ữ", "u", "ự", "u",
			"ý", "y", "ỳ", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
			"đ", "d",
		)
		normalized[index] = replacer.Replace(value)
	}
	return normalized
}

func csvValue(row []string, indexByName map[string]int, column string) string {
	index, ok := indexByName[column]
	if !ok || index >= len(row) {
		return ""
	}
	return row[index]
}

func cleanupExistingDemoData(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT id
		FROM order_history
		WHERE nguoi_tao_don = ?
	`, demoSeedUsername)
	if err != nil {
		return fmt.Errorf("select demo order history ids: %w", err)
	}
	defer rows.Close()

	orderHistoryIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan demo order history id: %w", err)
		}
		orderHistoryIDs = append(orderHistoryIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate demo order history ids: %w", err)
	}

	if len(orderHistoryIDs) > 0 {
		placeholders := makePlaceholders(len(orderHistoryIDs))
		args := make([]interface{}, 0, len(orderHistoryIDs))
		for _, id := range orderHistoryIDs {
			args = append(args, id)
		}

		deleteReconciliationQuery := fmt.Sprintf("DELETE FROM order_invoice_reconciliation WHERE order_history_id IN (%s)", placeholders)
		if _, err := tx.Exec(deleteReconciliationQuery, args...); err != nil {
			return fmt.Errorf("delete demo reconciliations: %w", err)
		}

		deleteOrderHistoryQuery := fmt.Sprintf("DELETE FROM order_history WHERE id IN (%s)", placeholders)
		if _, err := tx.Exec(deleteOrderHistoryQuery, args...); err != nil {
			return fmt.Errorf("delete demo order history: %w", err)
		}
	}

	if _, err := tx.Exec(`
		DELETE FROM order_invoice_reconciliation
		WHERE note LIKE 'demo export-to-vinmes%'
	`); err != nil {
		return fmt.Errorf("delete legacy demo reconciliations: %w", err)
	}

	return nil
}

func ensureHoaDonRow(tx *sql.Tx, row csvInvoiceRow) (hoaDonRow, error) {
	existing, err := findHoaDonRow(tx, row.IDHoaDon, row.STTDongHang)
	if err == nil {
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return hoaDonRow{}, err
	}

	result, err := tx.Exec(`
		INSERT INTO hoa_don (
			company_contact_id,
			trang_thai_hoa_don,
			loai_hoa_don,
			so_hoa_don,
			ngay_hoa_don,
			ma_so_thue_nguoi_ban,
			cong_ty,
			dia_chi,
			link_tra_cuu_hoa_don,
			id_hoa_don,
			stt_dong_hang,
			ten_hang_hoa,
			ma_hang_hoa,
			don_vi_tinh,
			so_luong,
			don_gia_chua_thue,
			thue_suat_gtgt
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nil,
		row.TrangThaiHoaDon,
		row.LoaiHoaDon,
		row.SoHoaDon,
		row.NgayHoaDon,
		row.MaSoThueNguoiBan,
		row.CongTy,
		row.DiaChi,
		row.LinkTraCuuHoaDon,
		row.IDHoaDon,
		row.STTDongHang,
		row.TenHangHoa,
		row.MaHangHoa,
		row.DonViTinh,
		row.SoLuong,
		row.DonGiaChuaThue,
		row.ThueSuatGTGT,
	)
	if err != nil {
		return hoaDonRow{}, fmt.Errorf("insert hoa_don row: %w", err)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return hoaDonRow{}, fmt.Errorf("read inserted hoa_don id: %w", err)
	}

	return hoaDonRow{ID: insertedID, csvInvoiceRow: row}, nil
}

func findHoaDonRow(tx *sql.Tx, invoiceID string, sttDongHang int) (hoaDonRow, error) {
	var row hoaDonRow
	err := tx.QueryRow(`
		SELECT
			id,
			trang_thai_hoa_don,
			loai_hoa_don,
			so_hoa_don,
			ngay_hoa_don,
			ma_so_thue_nguoi_ban,
			cong_ty,
			dia_chi,
			link_tra_cuu_hoa_don,
			id_hoa_don,
			stt_dong_hang,
			ten_hang_hoa,
			ma_hang_hoa,
			don_vi_tinh,
			so_luong,
			don_gia_chua_thue,
			thue_suat_gtgt
		FROM hoa_don
		WHERE id_hoa_don = ? AND stt_dong_hang = ?
		LIMIT 1
	`, invoiceID, sttDongHang).Scan(
		&row.ID,
		&row.TrangThaiHoaDon,
		&row.LoaiHoaDon,
		&row.SoHoaDon,
		&row.NgayHoaDon,
		&row.MaSoThueNguoiBan,
		&row.CongTy,
		&row.DiaChi,
		&row.LinkTraCuuHoaDon,
		&row.IDHoaDon,
		&row.STTDongHang,
		&row.TenHangHoa,
		&row.MaHangHoa,
		&row.DonViTinh,
		&row.SoLuong,
		&row.DonGiaChuaThue,
		&row.ThueSuatGTGT,
	)
	if err != nil {
		return hoaDonRow{}, err
	}
	return row, nil
}

func insertOrderHistory(tx *sql.Tx, actor *models.UserProfile, row hoaDonRow, now time.Time) (int64, error) {
	statusText := "Đã gửi email"
	nowText := now.Format(time.RFC3339)

	result, err := tx.Exec(`
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
		nil,
		nil,
		row.CongTy,
		"",
		row.MaHangHoa,
		row.TenHangHoa,
		row.MaHangHoa,
		"",
		row.DonViTinh,
		"",
		int(math.Round(row.SoLuong)),
		config.AppConfig.DefaultCompanyContactEmail,
		models.OrderSourceManual,
		nil,
		actor.Username,
		actor.Email,
		nowText,
		actor.ID,
		demoSeedUsername,
		actor.Email,
		nowText,
		nowText,
		statusText,
		1,
		actor.ID,
		actor.Username,
		actor.Email,
	)
	if err != nil {
		return 0, fmt.Errorf("insert order_history: %w", err)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted order_history id: %w", err)
	}

	return insertedID, nil
}

func makePlaceholders(count int) string {
	parts := make([]string, count)
	for index := range parts {
		parts[index] = "?"
	}
	return strings.Join(parts, ", ")
}
