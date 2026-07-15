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

	rows, err := db.Query("SELECT id, ID_quyetdinh, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU FROM mapping2 LIMIT 5")
	if err != nil {
		log.Fatalf("select error: %v", err)
	}
	defer rows.Close()

	fmt.Println("First 5 rows of mapping2 table:")
	for rows.Next() {
		var id int
		var idqd, gn, qcdg, qcgh, qctt, tkm, tt sql.NullString
		if err := rows.Scan(&id, &idqd, &gn, &qcdg, &qcgh, &qctt, &tkm, &tt); err != nil {
			log.Fatalf("scan error: %v", err)
		}
		fmt.Printf("ID: %d | ID_quyetdinh: %q | Group: %q | QCDG: %q | QCGH: %q | QCTT: %q | TKM: %q | TONGTHAU: %q\n",
			id, idqd.String, gn.String, qcdg.String, qcgh.String, qctt.String, tkm.String, tt.String)
	}
}
