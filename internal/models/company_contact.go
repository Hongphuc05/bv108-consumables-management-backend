package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"bv108-consumables-management-backend/config"
)

type CompanyContact struct {
	MaSoThue       string `json:"maSoThue"`
	TenCongTy      string `json:"tenCongTy"`
	SoHD           string `json:"soHd,omitempty"`
	NgayHD         string `json:"ngayHd,omitempty"`
	DiaChiCongTy   string `json:"diaChiCongTy,omitempty"`
	SoTKNganHang   string `json:"soTkNganHang,omitempty"`
	TenNganHang    string `json:"tenNganHang,omitempty"`
	ChiNhanh       string `json:"chiNhanh,omitempty"`
	QD             string `json:"qd,omitempty"`
	SoGoiThau      string `json:"soGoiThau,omitempty"`
	Gmail          string `json:"gmail"`
	ID             string `json:"id"`
	IdentityKey    string `json:"identityKey"`
	CompanyName    string `json:"companyName"`
	TaxID          string `json:"taxId,omitempty"`
	Email          string `json:"email"`
	ContractNumber string `json:"contractNumber,omitempty"`
	ContractDate   string `json:"contractDate,omitempty"`
	CompanyAddress string `json:"companyAddress,omitempty"`
	BankAccount    string `json:"bankAccount,omitempty"`
	BankName       string `json:"bankName,omitempty"`
	BankBranch     string `json:"bankBranch,omitempty"`
	DecisionNumber string `json:"decisionNumber,omitempty"`
	PackageNumber  string `json:"packageNumber,omitempty"`
}

type CompanyContactRepository struct {
	DB *sql.DB
}

func NewCompanyContactRepository(db *sql.DB) *CompanyContactRepository {
	return &CompanyContactRepository{DB: db}
}

func ResolveDefaultCompanyContactEmail() string {
	if config.AppConfig == nil {
		return ""
	}

	if email := normalizeCompanyEmail(config.AppConfig.DefaultCompanyContactEmail); email != "" {
		return email
	}

	return normalizeCompanyEmail(config.AppConfig.SMTPFrom)
}

func (r *CompanyContactRepository) EnsureSchema() error {
	statement := `
		CREATE TABLE IF NOT EXISTS company_contacts (
			ma_so_thue VARCHAR(50) NOT NULL,
			ten_cong_ty VARCHAR(255) NOT NULL,
			so_hd VARCHAR(100),
			ngay_hd DATE,
			dia_chi_cong_ty TEXT,
			so_tk_ngan_hang VARCHAR(100),
			ten_ngan_hang VARCHAR(255),
			chi_nhanh VARCHAR(255),
			qd VARCHAR(255),
			so_goi_thau VARCHAR(255),
			gmail VARCHAR(255),
			PRIMARY KEY (ma_so_thue)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring company contacts schema: %w", err)
	}

	return nil
}

func (r *CompanyContactRepository) SyncFromExistingData(defaultEmail string) error {
	normalizedDefaultEmail := normalizeCompanyEmail(defaultEmail)
	if normalizedDefaultEmail == "" {
		normalizedDefaultEmail = ResolveDefaultCompanyContactEmail()
	}

	exists, err := r.tableExists("hoa_don")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if _, err := r.DB.Exec(`
		INSERT INTO company_contacts (
			ma_so_thue,
			ten_cong_ty,
			gmail
		)
		SELECT DISTINCT
			TRIM(ma_so_thue_nguoi_ban),
			TRIM(cong_ty),
			?
		FROM hoa_don
		WHERE TRIM(COALESCE(ma_so_thue_nguoi_ban, '')) <> ''
		  AND TRIM(COALESCE(cong_ty, '')) <> ''
		ON DUPLICATE KEY UPDATE
			ten_cong_ty = CASE
				WHEN TRIM(COALESCE(company_contacts.ten_cong_ty, '')) = '' THEN VALUES(ten_cong_ty)
				ELSE company_contacts.ten_cong_ty
			END,
			gmail = CASE
				WHEN TRIM(COALESCE(company_contacts.gmail, '')) = '' THEN VALUES(gmail)
				ELSE company_contacts.gmail
			END
	`, normalizedDefaultEmail); err != nil {
		return fmt.Errorf("error syncing company contacts from hoa_don: %w", err)
	}

	return nil
}

func (r *CompanyContactRepository) BackfillOrderReferences() error {
	return r.backfillOrderEmails("pending_orders")
}

func (r *CompanyContactRepository) EnsureContactTx(tx *sql.Tx, companyName, taxID, email string) (sql.NullInt64, string, error) {
	if normalizedEmail := normalizeCompanyEmail(email); normalizedEmail != "" {
		return sql.NullInt64{}, normalizedEmail, nil
	}

	contact, err := r.getByCompanyNameTx(tx, companyName)
	if err != nil {
		return sql.NullInt64{}, "", err
	}
	if contact != nil && strings.TrimSpace(contact.Email) != "" {
		return sql.NullInt64{}, strings.TrimSpace(contact.Email), nil
	}

	return sql.NullInt64{}, ResolveDefaultCompanyContactEmail(), nil
}

func (r *CompanyContactRepository) backfillOrderEmails(tableName string) error {
	exists, err := r.tableExists(tableName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if _, err := r.DB.Exec(fmt.Sprintf(`
		UPDATE %s o
		JOIN company_contacts c
			ON LOWER(TRIM(o.nha_thau)) = LOWER(TRIM(c.ten_cong_ty))
		SET o.email = COALESCE(NULLIF(TRIM(c.gmail), ''), ?)
		WHERE TRIM(COALESCE(o.nha_thau, '')) <> ''
	`, tableName), ResolveDefaultCompanyContactEmail()); err != nil {
		return fmt.Errorf("error backfilling %s company emails: %w", tableName, err)
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

func normalizeCompanyEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func buildCompanyIdentityKey(companyName, taxID string) string {
	normalizedName := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(companyName)), " "))
	if normalizedName != "" {
		return "name:" + normalizedName
	}

	taxID = strings.ToUpper(strings.TrimSpace(taxID))
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
		SELECT
			ma_so_thue,
			ten_cong_ty,
			COALESCE(so_hd, ''),
			COALESCE(DATE_FORMAT(ngay_hd, '%Y-%m-%d'), ''),
			COALESCE(dia_chi_cong_ty, ''),
			COALESCE(so_tk_ngan_hang, ''),
			COALESCE(ten_ngan_hang, ''),
			COALESCE(chi_nhanh, ''),
			COALESCE(qd, ''),
			COALESCE(so_goi_thau, ''),
			COALESCE(gmail, '')
		FROM company_contacts
	`
	args := make([]interface{}, 0, 3)
	if trimmedKeyword != "" {
		searchPattern := "%" + trimmedKeyword + "%"
		query += "\tWHERE ten_cong_ty LIKE ? OR ma_so_thue LIKE ?\n"
		args = append(args, searchPattern, searchPattern)
	}
	query += "\tORDER BY ten_cong_ty ASC\n\tLIMIT ?\n"
	args = append(args, limit)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error searching company contacts: %w", err)
	}
	defer rows.Close()

	contacts := make([]CompanyContact, 0)
	for rows.Next() {
		contact, err := scanCompanyContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, *contact)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company contacts: %w", err)
	}

	return contacts, nil
}

func (r *CompanyContactRepository) GetByID(id int64) (*CompanyContact, error) {
	if id <= 0 {
		return nil, nil
	}

	query := `
		SELECT
			ma_so_thue,
			ten_cong_ty,
			COALESCE(so_hd, ''),
			COALESCE(DATE_FORMAT(ngay_hd, '%Y-%m-%d'), ''),
			COALESCE(dia_chi_cong_ty, ''),
			COALESCE(so_tk_ngan_hang, ''),
			COALESCE(ten_ngan_hang, ''),
			COALESCE(chi_nhanh, ''),
			COALESCE(qd, ''),
			COALESCE(so_goi_thau, ''),
			COALESCE(gmail, '')
		FROM company_contacts
		WHERE ma_so_thue = ?
		LIMIT 1
	`

	contact, err := scanCompanyContact(r.DB.QueryRow(query, strconv.FormatInt(id, 10)))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading company contact by id: %w", err)
	}

	return contact, nil
}

func (r *CompanyContactRepository) GetByTaxID(taxID string) (*CompanyContact, error) {
	normalizedTaxID := strings.TrimSpace(taxID)
	if normalizedTaxID == "" {
		return nil, nil
	}

	query := `
		SELECT
			ma_so_thue,
			ten_cong_ty,
			COALESCE(so_hd, ''),
			COALESCE(DATE_FORMAT(ngay_hd, '%Y-%m-%d'), ''),
			COALESCE(dia_chi_cong_ty, ''),
			COALESCE(so_tk_ngan_hang, ''),
			COALESCE(ten_ngan_hang, ''),
			COALESCE(chi_nhanh, ''),
			COALESCE(qd, ''),
			COALESCE(so_goi_thau, ''),
			COALESCE(gmail, '')
		FROM company_contacts
		WHERE ma_so_thue = ?
		LIMIT 1
	`

	contact, err := scanCompanyContact(r.DB.QueryRow(query, normalizedTaxID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading company contact by tax id: %w", err)
	}

	return contact, nil
}

func (r *CompanyContactRepository) GetByCompanyName(companyName string) (*CompanyContact, error) {
	return r.getByCompanyNameTx(nil, companyName)
}

func (r *CompanyContactRepository) getByCompanyNameTx(tx *sql.Tx, companyName string) (*CompanyContact, error) {
	normalizedName := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(companyName)), " "))
	if normalizedName == "" {
		return nil, nil
	}

	query := `
		SELECT
			ma_so_thue,
			ten_cong_ty,
			COALESCE(so_hd, ''),
			COALESCE(DATE_FORMAT(ngay_hd, '%Y-%m-%d'), ''),
			COALESCE(dia_chi_cong_ty, ''),
			COALESCE(so_tk_ngan_hang, ''),
			COALESCE(ten_ngan_hang, ''),
			COALESCE(chi_nhanh, ''),
			COALESCE(qd, ''),
			COALESCE(so_goi_thau, ''),
			COALESCE(gmail, '')
		FROM company_contacts
		WHERE LOWER(TRIM(ten_cong_ty)) = ?
		ORDER BY ma_so_thue ASC
	`

	var (
		rows *sql.Rows
		err  error
	)
	if tx != nil {
		rows, err = tx.Query(query, normalizedName)
	} else {
		rows, err = r.DB.Query(query, normalizedName)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading company contact by name: %w", err)
	}
	defer rows.Close()

	matches := make([]CompanyContact, 0, 2)
	for rows.Next() {
		contact, err := scanCompanyContact(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning company contact by name: %w", err)
		}
		matches = append(matches, *contact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company contact by name: %w", err)
	}

	return selectCompanyContactByNameMatches(matches), nil
}

func selectCompanyContactByNameMatches(matches []CompanyContact) *CompanyContact {
	if len(matches) == 0 {
		return nil
	}

	selected := matches[0]
	if len(matches) == 1 {
		return &selected
	}

	uniqueTaxIDs := make(map[string]struct{}, len(matches))
	sharedEmail := ""
	emailConflict := false

	for index, match := range matches {
		if taxID := strings.TrimSpace(match.MaSoThue); taxID != "" {
			uniqueTaxIDs[taxID] = struct{}{}
		}

		email := normalizeCompanyEmail(match.Email)
		if index == 0 {
			sharedEmail = email
			continue
		}

		if email != sharedEmail {
			emailConflict = true
		}
	}

	if len(uniqueTaxIDs) <= 1 {
		return &selected
	}

	// Multiple tax IDs share the same normalized company name.
	// Keep a shared email only when every duplicate resolves to the same address,
	// but avoid binding an arbitrary company_contact_id/tax ID.
	selected.MaSoThue = ""
	selected.ID = ""
	selected.IdentityKey = buildCompanyIdentityKey(selected.TenCongTy, "")
	selected.TaxID = ""

	if emailConflict {
		selected.Gmail = ""
		selected.Email = ""
	} else {
		selected.Gmail = sharedEmail
		selected.Email = sharedEmail
	}

	return &selected
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCompanyContact(row scanner) (*CompanyContact, error) {
	var contact CompanyContact
	if err := row.Scan(
		&contact.MaSoThue,
		&contact.TenCongTy,
		&contact.SoHD,
		&contact.NgayHD,
		&contact.DiaChiCongTy,
		&contact.SoTKNganHang,
		&contact.TenNganHang,
		&contact.ChiNhanh,
		&contact.QD,
		&contact.SoGoiThau,
		&contact.Gmail,
	); err != nil {
		return nil, err
	}

	if contact.Gmail == "" {
		contact.Gmail = ResolveDefaultCompanyContactEmail()
	}

	contact.ID = contact.MaSoThue
	contact.IdentityKey = buildCompanyIdentityKey(contact.TenCongTy, contact.MaSoThue)
	contact.CompanyName = contact.TenCongTy
	contact.TaxID = contact.MaSoThue
	contact.Email = contact.Gmail
	contact.ContractNumber = contact.SoHD
	contact.ContractDate = contact.NgayHD
	contact.CompanyAddress = contact.DiaChiCongTy
	contact.BankAccount = contact.SoTKNganHang
	contact.BankName = contact.TenNganHang
	contact.BankBranch = contact.ChiNhanh
	contact.DecisionNumber = contact.QD
	contact.PackageNumber = contact.SoGoiThau

	return &contact, nil
}

// ReplaceAll inserts/updates company contacts list in transaction
func (r *CompanyContactRepository) ReplaceAll(contacts []CompanyContact) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction for company contacts sync: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	insertSQL := `
		INSERT INTO company_contacts (
			ma_so_thue, ten_cong_ty, so_hd, ngay_hd, dia_chi_cong_ty,
			so_tk_ngan_hang, ten_ngan_hang, chi_nhanh, qd, so_goi_thau, gmail
		) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?)
		ON DUPLICATE KEY UPDATE
			ten_cong_ty = VALUES(ten_cong_ty),
			gmail = COALESCE(NULLIF(company_contacts.gmail, ''), VALUES(gmail))
	`
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("error preparing company contacts sync statement: %w", err)
	}
	defer stmt.Close()

	for _, c := range contacts {
		var ngayHD interface{} = c.NgayHD
		if c.NgayHD == "" {
			ngayHD = nil
		}
		if _, err = stmt.Exec(
			c.MaSoThue,
			c.TenCongTy,
			c.SoHD,
			ngayHD,
			c.DiaChiCongTy,
			c.SoTKNganHang,
			c.TenNganHang,
			c.ChiNhanh,
			c.QD,
			c.SoGoiThau,
			c.Gmail,
		); err != nil {
			return fmt.Errorf("error inserting company contact %q: %w", c.MaSoThue, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing company contacts sync: %w", err)
	}

	return nil
}

