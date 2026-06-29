package main

import (
	"fmt"
	"log"

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

	runCount("supplies", "SELECT COUNT(*) FROM supplies")
	runCount("forecast_approvals", "SELECT COUNT(*) FROM forecast_approvals")
	runCount("forecast_monthly_snapshots", "SELECT COUNT(*) FROM forecast_monthly_snapshots")
	runCount("pending_orders", "SELECT COUNT(*) FROM pending_orders")
	runCount("order_history", "SELECT COUNT(*) FROM order_history")
	runCount("order_invoice_reconciliation", "SELECT COUNT(*) FROM order_invoice_reconciliation")

	fmt.Println("")
	fmt.Println("Forecast approvals by period/status:")
	runPeriodStatusReport("forecast_approvals", `
		SELECT forecast_year, forecast_month, status, COUNT(*) AS total
		FROM forecast_approvals
		GROUP BY forecast_year, forecast_month, status
		ORDER BY forecast_year DESC, forecast_month DESC, status ASC
	`)

	fmt.Println("")
	fmt.Println("Forecast snapshots by period/status:")
	runPeriodStatusReport("forecast_monthly_snapshots", `
		SELECT forecast_year, forecast_month, status, COUNT(*) AS total
		FROM forecast_monthly_snapshots
		GROUP BY forecast_year, forecast_month, status
		ORDER BY forecast_year DESC, forecast_month DESC, status ASC
	`)

	fmt.Println("")
	fmt.Println("Recent reconciliation periods:")

	rows, err := database.DB.Query(`
		SELECT
			YEAR(COALESCE(invoice_time, matched_at)) AS y,
			MONTH(COALESCE(invoice_time, matched_at)) AS m,
			COUNT(*) AS total
		FROM order_invoice_reconciliation
		WHERE has_invoice = 1
		GROUP BY y, m
		ORDER BY y DESC, m DESC
		LIMIT 12
	`)
	if err != nil {
		log.Fatalf("query reconciliation periods: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var year, month, total int
		if err := rows.Scan(&year, &month, &total); err != nil {
			log.Fatalf("scan reconciliation periods: %v", err)
		}
		fmt.Printf("- %04d-%02d: %d rows\n", year, month, total)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("iterate reconciliation periods: %v", err)
	}
}

func runCount(label, query string) {
	var count int
	if err := database.DB.QueryRow(query).Scan(&count); err != nil {
		log.Fatalf("query %s count: %v", label, err)
	}
	fmt.Printf("%s: %d\n", label, count)
}

func runPeriodStatusReport(label, query string) {
	rows, err := database.DB.Query(query)
	if err != nil {
		log.Fatalf("query %s report: %v", label, err)
	}
	defer rows.Close()

	foundRows := false
	for rows.Next() {
		foundRows = true
		var year, month, total int
		var status string
		if err := rows.Scan(&year, &month, &status, &total); err != nil {
			log.Fatalf("scan %s report: %v", label, err)
		}
		fmt.Printf("- %04d-%02d [%s]: %d rows\n", year, month, status, total)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("iterate %s report: %v", label, err)
	}

	if !foundRows {
		fmt.Println("- no rows")
	}
}
