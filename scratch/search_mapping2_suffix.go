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

	var count2023, count7319 int
	err = db.QueryRow("SELECT COUNT(*) FROM mapping2 WHERE ID_quyetdinh LIKE '%2023%'").Scan(&count2023)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}
	err = db.QueryRow("SELECT COUNT(*) FROM mapping2 WHERE ID_quyetdinh LIKE '%7319%'").Scan(&count7319)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}

	fmt.Printf("Total rows in mapping2 with '2023': %d\n", count2023)
	fmt.Printf("Total rows in mapping2 with '7319': %d\n", count7319)

	// Print unique suffixes in mapping2
	rows, err := db.Query("SELECT DISTINCT SUBSTRING_INDEX(ID_quyetdinh, '_', -1) FROM mapping2")
	if err != nil {
		log.Fatalf("query distinct error: %v", err)
	}
	defer rows.Close()

	fmt.Println("Unique suffixes in mapping2:")
	for rows.Next() {
		var suffix string
		rows.Scan(&suffix)
		fmt.Printf("  * %s\n", suffix)
	}
}
