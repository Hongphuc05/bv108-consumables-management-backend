package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/models"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./scratch/import_supplies_csv.go <csv_path>")
	}

	csvPath := os.Args[1]

	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	inputs, err := loadCSV(csvPath)
	if err != nil {
		log.Fatalf("load csv: %v", err)
	}

	if err := replaceSupplies(database.DB, inputs); err != nil {
		log.Fatalf("replace supplies: %v", err)
	}

	fmt.Printf("Imported %d supplies from %s\n", len(inputs), csvPath)
}

func replaceSupplies(db *sql.DB, inputs []models.SupplyUpsertInput) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec("DELETE FROM supplies"); err != nil {
		return fmt.Errorf("delete old supplies: %w", err)
	}

	const chunkSize = 250
	baseSQL := `
		INSERT INTO supplies (
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT,
			QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, THONG_TIN_THAU, TONGTHAU,
			HANGSX, NUOC_SX, NHA_CUNG_CAP, PRICE, TONDAUKY, NHAPTRONGKY,
			XUATTRONGKY, TONGNHAP, TON_KHO_MIN
		) VALUES %s
	`

	for start := 0; start < len(inputs); start += chunkSize {
		end := start + chunkSize
		if end > len(inputs) {
			end = len(inputs)
		}

		var (
			valuePlaceholders []string
			args              []interface{}
		)
		for _, input := range inputs[start:end] {
			valuePlaceholders = append(valuePlaceholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				input.IDX1,
				input.ProductID,
				input.GroupName,
				input.ID,
				input.IDX2,
				input.MaHieu,
				input.TypeName,
				input.Name,
				input.Unit,
				input.QuyCachDongGoi,
				input.QuyCachGiaoHang,
				input.ThongTinThau,
				input.TongThau,
				input.HangSX,
				input.NuocSX,
				input.NhaCungCap,
				input.Price,
				input.TonDauKy,
				input.NhapTrongKy,
				input.XuatTrongKy,
				input.TongNhap,
				input.TonKhoMin,
			)
		}

		query := fmt.Sprintf(baseSQL, strings.Join(valuePlaceholders, ","))
		if _, err = tx.Exec(query, args...); err != nil {
			return fmt.Errorf("insert chunk %d-%d: %w", start, end, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func loadCSV(path string) ([]models.SupplyUpsertInput, error) {
	file, err := os.Open(path)
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

	headerIndex := make(map[string]int, len(rows[0]))
	for i, column := range rows[0] {
		headerIndex[normalizeHeader(column)] = i
	}

	required := []string{
		"IDX1", "PRODUCTID", "GROUPNAME", "ID", "IDX2", "MA_HIEU", "TYPENAME", "NAME", "UNIT",
		"QUY_CACH_DONG_GOI", "QUY_CACH_GIAO_HANG", "THONG_TIN_THAU", "TONGTHAU", "HANGSX",
		"NUOC_SX", "NHA_CUNG_CAP", "PRICE", "TONDAUKY", "NHAPTRONGKY", "XUATTRONGKY", "TONGNHAP", "TON_KHO_MIN",
	}
	for _, key := range required {
		if _, ok := headerIndex[key]; !ok {
			return nil, fmt.Errorf("missing required column %q", key)
		}
	}

	inputs := make([]models.SupplyUpsertInput, 0, len(rows)-1)
	for rowNum, row := range rows[1:] {
		if isEmptyRow(row) {
			continue
		}

		idx1, err := parseInt(getCell(row, headerIndex, "IDX1"))
		if err != nil {
			return nil, fmt.Errorf("row %d IDX1: %w", rowNum+2, err)
		}
		productID, err := parseInt(getCell(row, headerIndex, "PRODUCTID"))
		if err != nil {
			return nil, fmt.Errorf("row %d PRODUCTID: %w", rowNum+2, err)
		}
		price, err := parseFloat(getCell(row, headerIndex, "PRICE"))
		if err != nil {
			return nil, fmt.Errorf("row %d PRICE: %w", rowNum+2, err)
		}
		tonDauKy, err := parseInt(getCell(row, headerIndex, "TONDAUKY"))
		if err != nil {
			return nil, fmt.Errorf("row %d TONDAUKY: %w", rowNum+2, err)
		}
		nhapTrongKy, err := parseInt(getCell(row, headerIndex, "NHAPTRONGKY"))
		if err != nil {
			return nil, fmt.Errorf("row %d NHAPTRONGKY: %w", rowNum+2, err)
		}
		xuatTrongKy, err := parseInt(getCell(row, headerIndex, "XUATTRONGKY"))
		if err != nil {
			return nil, fmt.Errorf("row %d XUATTRONGKY: %w", rowNum+2, err)
		}
		tongNhap, err := parseInt(getCell(row, headerIndex, "TONGNHAP"))
		if err != nil {
			return nil, fmt.Errorf("row %d TONGNHAP: %w", rowNum+2, err)
		}
		tonKhoMin, err := parseInt(getCell(row, headerIndex, "TON_KHO_MIN"))
		if err != nil {
			return nil, fmt.Errorf("row %d TON_KHO_MIN: %w", rowNum+2, err)
		}

		inputs = append(inputs, models.SupplyUpsertInput{
			IDX1:            idx1,
			ProductID:       productID,
			GroupName:       getCell(row, headerIndex, "GROUPNAME"),
			ID:              getCell(row, headerIndex, "ID"),
			IDX2:            getCell(row, headerIndex, "IDX2"),
			MaHieu:          getCell(row, headerIndex, "MA_HIEU"),
			TypeName:        getCell(row, headerIndex, "TYPENAME"),
			Name:            getCell(row, headerIndex, "NAME"),
			Unit:            getCell(row, headerIndex, "UNIT"),
			QuyCachDongGoi:  getCell(row, headerIndex, "QUY_CACH_DONG_GOI"),
			QuyCachGiaoHang: getCell(row, headerIndex, "QUY_CACH_GIAO_HANG"),
			ThongTinThau:    getCell(row, headerIndex, "THONG_TIN_THAU"),
			TongThau:        getCell(row, headerIndex, "TONGTHAU"),
			HangSX:          getCell(row, headerIndex, "HANGSX"),
			NuocSX:          getCell(row, headerIndex, "NUOC_SX"),
			NhaCungCap:      getCell(row, headerIndex, "NHA_CUNG_CAP"),
			Price:           price,
			TonDauKy:        tonDauKy,
			NhapTrongKy:     nhapTrongKy,
			XuatTrongKy:     xuatTrongKy,
			TongNhap:        tongNhap,
			TonKhoMin:       tonKhoMin,
		})
	}

	return inputs, nil
}

func normalizeHeader(value string) string {
	replacer := strings.NewReplacer("\uFEFF", "", " ", "", "\t", "", "\r", "", "\n", "")
	return strings.ToUpper(strings.TrimSpace(replacer.Replace(value)))
}

func getCell(row []string, index map[string]int, key string) string {
	position, ok := index[key]
	if !ok || position >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[position])
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseInt(raw string) (int, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if normalized == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(normalized)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseFloat(raw string) (float64, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if normalized == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
