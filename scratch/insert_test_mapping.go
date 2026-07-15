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

	// Insert test mapping for D00071
	query := `
		INSERT INTO mapping2 (ID_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query,
		"D00071_7319/QD-BV;G1;N1;2023",
		"Nhóm thử nghiệm Axit Etchinh",
		"10 Hộp/Thùng",
		"1 Thùng",
		"1 Hộp",
		"50",
		"1000",
	)
	if err != nil {
		log.Fatalf("insert error: %v", err)
	}

	fmt.Println("Successfully inserted test mapping for D00071 in mapping2 table!")
}
