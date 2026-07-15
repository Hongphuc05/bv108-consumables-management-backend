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

	var gn, qcdg, qcgh, qctt, tkm, tt sql.NullString
	err = db.QueryRow("SELECT GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU FROM supplies WHERE ID = 'D00071'").Scan(&gn, &qcdg, &qcgh, &qctt, &tkm, &tt)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}

	fmt.Println("Current DB values for supply D00071:")
	fmt.Printf("  * GROUPNAME: %q\n", gn.String)
	fmt.Printf("  * QUY_CACH_DONG_GOI: %q\n", qcdg.String)
	fmt.Printf("  * QUY_CACH_GIAO_HANG: %q\n", qcgh.String)
	fmt.Printf("  * QUY_CACH_TOI_THIEU: %q\n", qctt.String)
	fmt.Printf("  * TON_KHO_MIN: %q\n", tkm.String)
	fmt.Printf("  * TONGTHAU: %q\n", tt.String)
}
