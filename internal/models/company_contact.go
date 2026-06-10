package models

import (
	"database/sql"
	"fmt"
	"strings"
)

const DefaultCompanyContactEmail = "ngottha110@gmail.com"

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

	return sql.NullInt64{}, DefaultCompanyContactEmail, nil
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
	`, tableName), DefaultCompanyContactEmail); err != nil {
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
	return nil, nil
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
		LIMIT 1
	`

	var row scanner
	if tx != nil {
		row = tx.QueryRow(query, normalizedName)
	} else {
		row = r.DB.QueryRow(query, normalizedName)
	}

	contact, err := scanCompanyContact(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading company contact by name: %w", err)
	}

	return contact, nil
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
		contact.Gmail = DefaultCompanyContactEmail
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
