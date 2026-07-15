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

	rows, err := db.Query("SELECT id, ID_quyetdinh, GROUPNAME, TONGTHAU FROM mapping2 WHERE ID_quyetdinh LIKE '%D00071%'")
	if err != nil {
		log.Fatalf("select error: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var idqd, gn, tt sql.NullString
		rows.Scan(&id, &idqd, &gn, &tt)
		fmt.Printf("ID: %d | ID_quyetdinh: %q | Group: %q | TONGTHAU: %q\n", id, idqd.String, gn.String, tt.String)
		count++
	}
	fmt.Printf("Total matches for D00071 in mapping2: %d\n", count)
}
