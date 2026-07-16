package models

import (
	"database/sql"
	"fmt"
	"time"
)

type VinmesCatalogItem struct {
	CatalogType string
	ExternalID  string
	Code        string
	Name        string
	TaxCode     string
	BankAccount string
	TaxRate     *float64
	RawPayload  string
	SyncedAt    time.Time
}

type VinmesCatalogRepository struct {
	DB *sql.DB
}

func NewVinmesCatalogRepository(db *sql.DB) *VinmesCatalogRepository {
	return &VinmesCatalogRepository{DB: db}
}

func (r *VinmesCatalogRepository) EnsureSchema() error {
	_, err := r.DB.Exec(`
		CREATE TABLE IF NOT EXISTS vinmes_catalog_items (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			catalog_type VARCHAR(50) NOT NULL,
			external_id VARCHAR(255) NOT NULL,
			code VARCHAR(255) NULL,
			name TEXT NULL,
			tax_code VARCHAR(100) NULL,
			bank_account VARCHAR(255) NULL,
			tax_rate DECIMAL(12,4) NULL,
			raw_payload LONGTEXT NOT NULL,
			synced_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_vinmes_catalog_type_external (catalog_type, external_id),
			KEY idx_vinmes_catalog_type_code (catalog_type, code),
			KEY idx_vinmes_catalog_type_tax (catalog_type, tax_code),
			KEY idx_vinmes_catalog_type_bank (catalog_type, bank_account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("error ensuring Vinmes catalog schema: %w", err)
	}

	// Vinmes can return duplicate external IDs, so preserve every source row.
	var uniqueIndexCount int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
			AND table_name = 'vinmes_catalog_items'
			AND index_name = 'uq_vinmes_catalog_type_external'
	`).Scan(&uniqueIndexCount); err != nil {
		return fmt.Errorf("error checking legacy Vinmes catalog index: %w", err)
	}
	if uniqueIndexCount > 0 {
		if _, err := r.DB.Exec(`
			ALTER TABLE vinmes_catalog_items
				DROP INDEX uq_vinmes_catalog_type_external,
				ADD INDEX idx_vinmes_catalog_type_external (catalog_type, external_id)
		`); err != nil {
			return fmt.Errorf("error migrating Vinmes catalog external ID index: %w", err)
		}
	}
	return nil
}

func (r *VinmesCatalogRepository) ReplaceAll(items []VinmesCatalogItem, syncedAt time.Time) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting Vinmes catalog transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM vinmes_catalog_items"); err != nil {
		return fmt.Errorf("error clearing Vinmes catalog: %w", err)
	}

	statement, err := tx.Prepare(`
		INSERT INTO vinmes_catalog_items (
			catalog_type, external_id, code, name, tax_code,
			bank_account, tax_rate, raw_payload, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("error preparing Vinmes catalog insert: %w", err)
	}
	defer statement.Close()

	for _, item := range items {
		if _, err := statement.Exec(
			item.CatalogType,
			item.ExternalID,
			nullableCatalogString(item.Code),
			nullableCatalogString(item.Name),
			nullableCatalogString(item.TaxCode),
			nullableCatalogString(item.BankAccount),
			item.TaxRate,
			item.RawPayload,
			syncedAt,
		); err != nil {
			return fmt.Errorf("error inserting Vinmes %s item %s: %w", item.CatalogType, item.ExternalID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing Vinmes catalog refresh: %w", err)
	}
	return nil
}

func (r *VinmesCatalogRepository) ListAll() ([]VinmesCatalogItem, error) {
	rows, err := r.DB.Query(`
		SELECT
			catalog_type,
			external_id,
			COALESCE(code, ''),
			COALESCE(name, ''),
			COALESCE(tax_code, ''),
			COALESCE(bank_account, ''),
			tax_rate,
			raw_payload,
			synced_at
		FROM vinmes_catalog_items
		ORDER BY catalog_type, id
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing Vinmes catalog: %w", err)
	}
	defer rows.Close()

	items := make([]VinmesCatalogItem, 0)
	for rows.Next() {
		var item VinmesCatalogItem
		var taxRate sql.NullFloat64
		if err := rows.Scan(
			&item.CatalogType,
			&item.ExternalID,
			&item.Code,
			&item.Name,
			&item.TaxCode,
			&item.BankAccount,
			&taxRate,
			&item.RawPayload,
			&item.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning Vinmes catalog item: %w", err)
		}
		if taxRate.Valid {
			value := taxRate.Float64
			item.TaxRate = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Vinmes catalog: %w", err)
	}
	return items, nil
}

func nullableCatalogString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
