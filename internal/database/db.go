package database

import (
	"database/sql"
	"fmt"
	"log"

	"bv108-backend/config"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// InitDB initializes database connection
func InitDB() error {
	var err error

	dsn := config.AppConfig.GetDSN()
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	log.Println("✅ Database connection established successfully")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}
