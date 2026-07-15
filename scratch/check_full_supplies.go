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

	var exists bool
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'hospital_db' AND table_name = 'full_supplies'").Scan(&exists)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}

	if !exists {
		fmt.Println("Table full_supplies does NOT exist in hospital_db!")
		return
	}

	fmt.Println("Table full_supplies exists! Describing structure:")
	rows, err := db.Query("DESCRIBE full_supplies")
	if err != nil {
		log.Fatalf("describe error: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var field, colType, null, key, extra string
		var def sql.NullString
		rows.Scan(&field, &colType, &null, &key, &def, &extra)
		fmt.Printf("  * %s (%s, Key: %s)\n", field, colType, key)
	}
}
