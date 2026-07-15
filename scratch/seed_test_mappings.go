package main

import (
	"database/sql"
	"fmt"
	"log"

	"bv108-consumables-management-backend/config"
	_ "github.com/go-sql-driver/mysql"
)

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

	// Clear old test mappings
	db.Exec("DELETE FROM mapping WHERE typename_quyetdinh LIKE 'N99.99.999%'")
	db.Exec("DELETE FROM mapping2 WHERE ID_quyetdinh LIKE 'TEST%'")

	// 1. Insert into mapping table (typename_quyetdinh)
	query1 := `
		INSERT INTO mapping (typename_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, tongthau)
		VALUES 
		('N99.99.999-999-991_TEST_QD_001', 'Nhóm Test 1 (Bảng mapping)', '10 Cái/Hộp', '1 Hộp', '1 Cái', '10', '100'),
		('N99.99.999-999-992_TEST_QD_002', 'Nhóm Test 2 (Bảng mapping)', '20 Hộp/Thùng', '1 Thùng', '1 Hộp', '20', '200'),
		('N99.99.999-999-993_TEST_QD_003', 'Nhóm Test 3 (Bảng mapping)', '50 Bộ/Kiện', '1 Kiện', '1 Bộ', '30', '300')
	`
	_, err = db.Exec(query1)
	if err != nil {
		log.Fatalf("insert mapping error: %v", err)
	}
	fmt.Println("Inserted test mappings into mapping table successfully!")

	// 2. Insert into mapping2 table (ID_quyetdinh)
	query2 := `
		INSERT INTO mapping2 (ID_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU)
		VALUES 
		('TEST001_TEST_QD_001', 'Nhóm Test 1 (Bảng mapping2)', '10 Cái/Hộp', '1 Hộp', '1 Cái', '10', '100'),
		('TEST002_TEST_QD_002', 'Nhóm Test 2 (Bảng mapping2)', '20 Hộp/Thùng', '1 Thùng', '1 Hộp', '20', '200'),
		('TEST003_TEST_QD_003', 'Nhóm Test 3 (Bảng mapping2)', '50 Bộ/Kiện', '1 Kiện', '1 Bộ', '30', '300')
	`
	_, err = db.Exec(query2)
	if err != nil {
		log.Fatalf("insert mapping2 error: %v", err)
	}
	fmt.Println("Inserted test mappings into mapping2 table successfully!")
}
