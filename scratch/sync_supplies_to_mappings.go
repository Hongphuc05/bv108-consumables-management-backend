package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"bv108-consumables-management-backend/config"
	_ "github.com/go-sql-driver/mysql"
)

type SupplyRow struct {
	ID              string
	TypeName        string
	GroupName       string
	QuyCachDongGoi  string
	QuyCachGiaoHang string
	QuyCachToiThieu string
	TonKhoMin       int
	TongThau        string
	ThongTinThau    string
}

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := config.AppConfig.GetDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// 1. Delete all data from mapping and mapping2
	fmt.Println("Clearing mapping table...")
	if _, err := db.Exec("DELETE FROM mapping"); err != nil {
		log.Fatalf("error clearing mapping table: %v", err)
	}
	fmt.Println("Clearing mapping2 table...")
	if _, err := db.Exec("DELETE FROM mapping2"); err != nil {
		log.Fatalf("error clearing mapping2 table: %v", err)
	}

	// 2. Select all supplies from full_supplies
	fmt.Println("Fetching all supplies from full_supplies...")
	suppliesQuery := `
		SELECT ID, TYPENAME, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU, THONG_TIN_THAU 
		FROM full_supplies
	`
	rows3, err := db.Query(suppliesQuery)
	if err != nil {
		log.Fatalf("query supplies error: %v", err)
	}
	defer rows3.Close()

	var supplies []SupplyRow
	for rows3.Next() {
		var s SupplyRow
		var id, tn, gn, qcdg, qcgh, qctt, tt, ttt sql.NullString
		var tkm int
		if err := rows3.Scan(&id, &tn, &gn, &qcdg, &qcgh, &qctt, &tkm, &tt, &ttt); err != nil {
			log.Fatalf("scan supply error: %v", err)
		}
		s.ID = id.String
		s.TypeName = tn.String
		s.GroupName = gn.String
		s.QuyCachDongGoi = qcdg.String
		s.QuyCachGiaoHang = qcgh.String
		s.QuyCachToiThieu = qctt.String
		s.TonKhoMin = tkm
		s.TongThau = tt.String
		s.ThongTinThau = ttt.String
		supplies = append(supplies, s)
	}
	fmt.Printf("Total supplies to process from full_supplies: %d\n", len(supplies))

	// Sets to track already inserted keys
	insertedMapping := make(map[string]bool)
	insertedMapping2 := make(map[string]bool)

	insertedMappingCount := 0
	insertedMapping2Count := 0

	for _, s := range supplies {
		// Only sync if thong_tin_thau is not empty
		if strings.TrimSpace(s.ThongTinThau) == "" {
			continue
		}

		// A. Process mapping table
		if strings.TrimSpace(s.TypeName) != "" {
			key := cleanMappingKey(s.TypeName + "_" + s.ThongTinThau)
			tonKhoMinStr := fmt.Sprintf("%d", s.TonKhoMin)

			if !insertedMapping[key] {
				insertQuery := `
					INSERT INTO mapping (typename_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, tongthau)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`
				standardKey := strings.ReplaceAll(s.TypeName+"_"+s.ThongTinThau, "Đ", "D")
				standardKey = strings.ReplaceAll(standardKey, "đ", "d")
				_, err = db.Exec(insertQuery, standardKey, s.GroupName, s.QuyCachDongGoi, s.QuyCachGiaoHang, s.QuyCachToiThieu, tonKhoMinStr, s.TongThau)
				if err == nil {
					insertedMappingCount++
					insertedMapping[key] = true
				}
			}
		}

		// B. Process mapping2 table
		if strings.TrimSpace(s.ID) != "" {
			key := cleanMappingKey(s.ID + "_" + s.ThongTinThau)
			tonKhoMinStr := fmt.Sprintf("%d", s.TonKhoMin)

			if !insertedMapping2[key] {
				insertQuery := `
					INSERT INTO mapping2 (ID_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`
				standardKey := strings.ReplaceAll(s.ID+"_"+s.ThongTinThau, "Đ", "D")
				standardKey = strings.ReplaceAll(standardKey, "đ", "d")
				_, err = db.Exec(insertQuery, standardKey, s.GroupName, s.QuyCachDongGoi, s.QuyCachGiaoHang, s.QuyCachToiThieu, tonKhoMinStr, s.TongThau)
				if err == nil {
					insertedMapping2Count++
					insertedMapping2[key] = true
				}
			}
		}
	}

	// 5. Append fake test mapping rows so they are always present
	fmt.Println("\nAppending fake test mappings...")
	db.Exec(`
		INSERT INTO mapping (typename_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, tongthau)
		VALUES 
		('N99.99.999-999-991_TEST_QD_001', 'Nhóm Test 1 (Bảng mapping)', '10 Cái/Hộp', '1 Hộp', '1 Cái', '10', '100'),
		('N99.99.999-999-992_TEST_QD_002', 'Nhóm Test 2 (Bảng mapping)', '20 Hộp/Thùng', '1 Thùng', '1 Hộp', '20', '200'),
		('N99.99.999-999-993_TEST_QD_003', 'Nhóm Test 3 (Bảng mapping)', '50 Bộ/Kiện', '1 Kiện', '1 Bộ', '30', '300')
	`)
	db.Exec(`
		INSERT INTO mapping2 (ID_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU)
		VALUES 
		('TEST001_TEST_QD_001', 'Nhóm Test 1 (Bảng mapping2)', '10 Cái/Hộp', '1 Hộp', '1 Cái', '10', '100'),
		('TEST002_TEST_QD_002', 'Nhóm Test 2 (Bảng mapping2)', '20 Hộp/Thùng', '1 Thùng', '1 Hộp', '20', '200'),
		('TEST003_TEST_QD_003', 'Nhóm Test 3 (Bảng mapping2)', '50 Bộ/Kiện', '1 Kiện', '1 Bộ', '30', '300')
	`)

	fmt.Println("\n=== SYNC SUMMARY (REPLACE ALL FROM full_supplies) ===")
	fmt.Printf("Mapping table:  Inserted %d unique rows\n", insertedMappingCount)
	fmt.Printf("Mapping2 table: Inserted %d unique rows\n", insertedMapping2Count)
}

func cleanMappingKey(key string) string {
	key = strings.ReplaceAll(key, "Đ", "D")
	key = strings.ReplaceAll(key, "đ", "d")
	return strings.TrimSpace(key)
}
