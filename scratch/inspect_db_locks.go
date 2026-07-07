package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	if len(os.Args) == 3 && os.Args[1] == "kill" {
		threadID, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("parse thread id: %v", err)
		}
		if _, err := database.DB.Exec(fmt.Sprintf("KILL %d", threadID)); err != nil {
			log.Fatalf("kill thread %d: %v", threadID, err)
		}
		fmt.Printf("Killed thread %d\n", threadID)
		return
	}

	fmt.Println("PROCESSLIST:")
	rows, err := database.DB.Query(`
		SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, INFO
		FROM information_schema.PROCESSLIST
		ORDER BY TIME DESC
	`)
	if err != nil {
		log.Fatalf("query processlist: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id      int64
			user    sql.NullString
			host    sql.NullString
			dbName  sql.NullString
			command sql.NullString
			sec     sql.NullInt64
			state   sql.NullString
			info    sql.NullString
		)
		if err := rows.Scan(&id, &user, &host, &dbName, &command, &sec, &state, &info); err != nil {
			log.Fatalf("scan processlist: %v", err)
		}
		fmt.Printf("- id=%d user=%q host=%q db=%q cmd=%q time=%d state=%q info=%q\n",
			id, user.String, host.String, dbName.String, command.String, sec.Int64, state.String, info.String)
	}

	fmt.Println("\nINNODB TRX:")
	trxRows, err := database.DB.Query(`
		SELECT trx_id, trx_state, trx_started, trx_mysql_thread_id, trx_query
		FROM information_schema.INNODB_TRX
		ORDER BY trx_started
	`)
	if err != nil {
		log.Fatalf("query innodb_trx: %v", err)
	}
	defer trxRows.Close()

	for trxRows.Next() {
		var (
			trxID    sql.NullString
			state    sql.NullString
			started  sql.NullString
			threadID sql.NullInt64
			query    sql.NullString
		)
		if err := trxRows.Scan(&trxID, &state, &started, &threadID, &query); err != nil {
			log.Fatalf("scan trx: %v", err)
		}
		fmt.Printf("- trx_id=%q state=%q started=%q thread_id=%d query=%q\n",
			trxID.String, state.String, started.String, threadID.Int64, query.String)
	}
}
