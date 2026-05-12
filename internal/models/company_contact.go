package models

import (
	"database/sql"
	"fmt"
	"strings"
)

const DefaultCompanyContactEmail = "ngottha110@gmail.com"

type CompanyContact struct {
	ID          int64  `json:"id"`
	IdentityKey string `json:"identityKey"`
	CompanyName string `json:"companyName"`
	TaxID       string `json:"taxId,omitempty"`
	Email       string `json:"email"`
}

type CompanyContactRepository struct {
	DB *sql.DB
}

func NewCompanyContactRepository(db *sql.DB) *CompanyContactRepository {
	return &CompanyContactRepository{DB: db}
}

func (r *CompanyContactRepository) EnsureSchema() error {
	statement := `
		CREATE TABLE IF NOT EXISTS company_contacts (
			id BIGINT NOT NULL AUTO_INCREMENT,
			identity_key VARCHAR(320) NOT NULL,
			company_name VARCHAR(255) NOT NULL,
			tax_id VARCHAR(50) NOT NULL DEFAULT '',
			email VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_company_contacts_identity (identity_key),
			KEY idx_company_contacts_tax_id (tax_id),
			KEY idx_company_contacts_company_name (company_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring company contacts schema: %w", err)
	}

	if err := r.ensureOrderRelationColumn("pending_orders", "id"); err != nil {
		return err
	}

	if err := r.ensureOrderRelationColumn("order_history", "pending_order_id"); err != nil {
		return err
	}

	return nil
}

func (r *CompanyContactRepository) SyncFromExistingData(defaultEmail string) error {
	sources := []struct {
		table string
		query string
	}{
		{
			table: "so_sanh_vat_tu",
			query: `
				SELECT DISTINCT
					TRIM(ten_cong_ty) AS company_name,
					TRIM(COALESCE(ma_so_thue, '')) AS tax_id
				FROM so_sanh_vat_tu
				WHERE ten_cong_ty IS NOT NULL AND TRIM(ten_cong_ty) <> ''
			`,
		},
		{
			table: "supplies",
			query: `
				SELECT DISTINCT
					TRIM(NHA_CUNG_CAP) AS company_name,
					'' AS tax_id
				FROM supplies
				WHERE NHA_CUNG_CAP IS NOT NULL AND TRIM(NHA_CUNG_CAP) <> ''
			`,
		},
		{
			table: "hoa_don",
			query: `
				SELECT DISTINCT
					TRIM(cong_ty) AS company_name,
					TRIM(COALESCE(ma_so_thue_nguoi_ban, '')) AS tax_id
				FROM hoa_don
				WHERE cong_ty IS NOT NULL AND TRIM(cong_ty) <> ''
			`,
		},
		{
			table: "pending_orders",
			query: `
				SELECT DISTINCT
					TRIM(nha_thau) AS company_name,
					'' AS tax_id
				FROM pending_orders
				WHERE nha_thau IS NOT NULL AND TRIM(nha_thau) <> ''
			`,
		},
		{
			table: "order_history",
			query: `
				SELECT DISTINCT
					TRIM(nha_thau) AS company_name,
					'' AS tax_id
				FROM order_history
				WHERE nha_thau IS NOT NULL AND TRIM(nha_thau) <> ''
			`,
		},
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting company contacts sync: %w", err)
	}
	defer tx.Rollback()

	for _, source := range sources {
		exists, err := r.tableExists(source.table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		if err := r.seedContactsFromQueryTx(tx, source.query, defaultEmail); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing company contacts sync: %w", err)
	}

	return nil
}

func (r *CompanyContactRepository) BackfillOrderReferences() error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting company contact backfill: %w", err)
	}
	defer tx.Rollback()

	if err := r.backfillOrderTableTx(tx, "pending_orders"); err != nil {
		return err
	}

	if err := r.backfillOrderTableTx(tx, "order_history"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing company contact backfill: %w", err)
	}

	return nil
}

func (r *CompanyContactRepository) EnsureContactTx(tx *sql.Tx, companyName, taxID, email string) (sql.NullInt64, string, error) {
	companyName = strings.Join(strings.Fields(strings.TrimSpace(companyName)), " ")
	taxID = normalizeCompanyTaxID(taxID)
	if companyName == "" && taxID == "" {
		return sql.NullInt64{}, "", nil
	}

	identityKey := buildCompanyIdentityKey(companyName, taxID)
	incomingEmail := normalizeCompanyEmail(email)
	defaultEmail := normalizeCompanyEmail(DefaultCompanyContactEmail)
	if incomingEmail == "" {
		incomingEmail = defaultEmail
	}

	var contact CompanyContact
	err := tx.QueryRow(`
		SELECT id, identity_key, company_name, tax_id, email
		FROM company_contacts
		WHERE identity_key = ?
		LIMIT 1
	`, identityKey).Scan(
		&contact.ID,
		&contact.IdentityKey,
		&contact.CompanyName,
		&contact.TaxID,
		&contact.Email,
	)
	if err == sql.ErrNoRows {
		result, execErr := tx.Exec(`
			INSERT INTO company_contacts (identity_key, company_name, tax_id, email)
			VALUES (?, ?, ?, ?)
		`, identityKey, companyName, taxID, incomingEmail)
		if execErr != nil {
			return sql.NullInt64{}, "", fmt.Errorf("error inserting company contact: %w", execErr)
		}

		contactID, execErr := result.LastInsertId()
		if execErr != nil {
			return sql.NullInt64{}, "", fmt.Errorf("error reading company contact id: %w", execErr)
		}

		return sql.NullInt64{Int64: contactID, Valid: true}, incomingEmail, nil
	}
	if err != nil {
		return sql.NullInt64{}, "", fmt.Errorf("error finding company contact: %w", err)
	}

	nextCompanyName := contact.CompanyName
	if nextCompanyName == "" {
		nextCompanyName = companyName
	}

	nextTaxID := contact.TaxID
	if nextTaxID == "" {
		nextTaxID = taxID
	}

	nextEmail := normalizeCompanyEmail(contact.Email)
	if incomingEmail != "" && incomingEmail != defaultEmail {
		nextEmail = incomingEmail
	} else if nextEmail == "" {
		nextEmail = incomingEmail
	}

	if nextCompanyName != contact.CompanyName || nextTaxID != contact.TaxID || nextEmail != normalizeCompanyEmail(contact.Email) {
		if _, err := tx.Exec(`
			UPDATE company_contacts
			SET company_name = ?, tax_id = ?, email = ?
			WHERE id = ?
		`, nextCompanyName, nextTaxID, nextEmail, contact.ID); err != nil {
			return sql.NullInt64{}, "", fmt.Errorf("error updating company contact: %w", err)
		}
	}

	return sql.NullInt64{Int64: contact.ID, Valid: true}, nextEmail, nil
}

func (r *CompanyContactRepository) backfillOrderTableTx(tx *sql.Tx, tableName string) error {
	rows, err := tx.Query(fmt.Sprintf(`
		SELECT id, nha_thau, email
		FROM %s
	`, tableName))
	if err != nil {
		return fmt.Errorf("error querying %s for company contact backfill: %w", tableName, err)
	}

	type orderRow struct {
		ID      int64
		Company string
		Email   string
	}

	records := make([]orderRow, 0)
	for rows.Next() {
		var record orderRow
		if err := rows.Scan(&record.ID, &record.Company, &record.Email); err != nil {
			return fmt.Errorf("error scanning %s row for company contact backfill: %w", tableName, err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating %s for company contact backfill: %w", tableName, err)
	}

	if err := rows.Close(); err != nil {
		return fmt.Errorf("error closing %s rows for company contact backfill: %w", tableName, err)
	}

	for _, record := range records {
		contactID, resolvedEmail, err := r.EnsureContactTx(tx, record.Company, "", record.Email)
		if err != nil {
			return err
		}
		if !contactID.Valid {
			continue
		}

		if _, err := tx.Exec(
			fmt.Sprintf("UPDATE %s SET company_contact_id = ?, email = ? WHERE id = ?", tableName),
			contactID.Int64,
			resolvedEmail,
			record.ID,
		); err != nil {
			return fmt.Errorf("error updating %s with company contact: %w", tableName, err)
		}
	}

	return nil
}

func (r *CompanyContactRepository) seedContactsFromQueryTx(tx *sql.Tx, query, defaultEmail string) error {
	rows, err := tx.Query(query)
	if err != nil {
		return fmt.Errorf("error querying source companies: %w", err)
	}

	type sourceCompany struct {
		CompanyName string
		TaxID       string
	}

	items := make([]sourceCompany, 0)

	for rows.Next() {
		var item sourceCompany
		if err := rows.Scan(&item.CompanyName, &item.TaxID); err != nil {
			return fmt.Errorf("error scanning source company: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating source companies: %w", err)
	}

	if err := rows.Close(); err != nil {
		return fmt.Errorf("error closing source companies rows: %w", err)
	}

	for _, item := range items {
		if _, _, err := r.EnsureContactTx(tx, item.CompanyName, item.TaxID, defaultEmail); err != nil {
			return err
		}
	}

	return nil
}

func (r *CompanyContactRepository) ensureOrderRelationColumn(tableName, afterColumn string) error {
	exists, err := r.tableExists(tableName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if err := r.ensureColumnExists(tableName, "company_contact_id", fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN company_contact_id BIGINT NULL AFTER %s",
		tableName,
		afterColumn,
	)); err != nil {
		return err
	}

	if err := r.ensureIndexExists(tableName, fmt.Sprintf("idx_%s_company_contact", tableName), fmt.Sprintf(
		"ALTER TABLE %s ADD INDEX idx_%s_company_contact (company_contact_id)",
		tableName,
		tableName,
	)); err != nil {
		return err
	}

	return nil
}

func (r *CompanyContactRepository) tableExists(tableName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?
	`, tableName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking table %s: %w", tableName, err)
	}

	return count > 0, nil
}

func (r *CompanyContactRepository) ensureColumnExists(tableName, columnName, alterStatement string) error {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, tableName, columnName).Scan(&count); err != nil {
		return fmt.Errorf("error checking column %s.%s: %w", tableName, columnName, err)
	}

	if count > 0 {
		return nil
	}

	if _, err := r.DB.Exec(alterStatement); err != nil {
		return fmt.Errorf("error altering %s.%s: %w", tableName, columnName, err)
	}

	return nil
}

func (r *CompanyContactRepository) ensureIndexExists(tableName, indexName, alterStatement string) error {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
	`, tableName, indexName).Scan(&count); err != nil {
		return fmt.Errorf("error checking index %s on %s: %w", indexName, tableName, err)
	}

	if count > 0 {
		return nil
	}

	if _, err := r.DB.Exec(alterStatement); err != nil {
		return fmt.Errorf("error creating index %s on %s: %w", indexName, tableName, err)
	}

	return nil
}

func normalizeCompanyEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeCompanyTaxID(taxID string) string {
	return strings.ToUpper(strings.TrimSpace(taxID))
}

func buildCompanyIdentityKey(companyName, taxID string) string {
	normalizedName := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(companyName)), " "))
	if normalizedName != "" {
		return "name:" + normalizedName
	}

	taxID = normalizeCompanyTaxID(taxID)
	if taxID != "" {
		return "tax:" + taxID
	}

	return ""
}

func (r *CompanyContactRepository) Search(keyword string, limit int) ([]CompanyContact, error) {
	trimmedKeyword := strings.TrimSpace(keyword)

	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	query := `
		SELECT id, identity_key, company_name, tax_id, email
		FROM company_contacts
	`
	args := make([]interface{}, 0, 3)
	if trimmedKeyword != "" {
		searchPattern := "%" + trimmedKeyword + "%"
		query += "\tWHERE company_name LIKE ? OR tax_id LIKE ?\n"
		args = append(args, searchPattern, searchPattern)
	}
	query += "\tORDER BY company_name ASC\n\tLIMIT ?\n"
	args = append(args, limit)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error searching company contacts: %w", err)
	}
	defer rows.Close()

	contacts := make([]CompanyContact, 0)
	for rows.Next() {
		var contact CompanyContact
		if err := rows.Scan(&contact.ID, &contact.IdentityKey, &contact.CompanyName, &contact.TaxID, &contact.Email); err != nil {
			return nil, fmt.Errorf("error scanning company contact: %w", err)
		}

		if contact.Email == "" {
			contact.Email = DefaultCompanyContactEmail
		}

		contacts = append(contacts, contact)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company contacts: %w", err)
	}

	return contacts, nil
}
