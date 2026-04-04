package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type SchemaMaintenanceRepository struct {
	DB *sql.DB
}

const relationalIntegrityMigrationKey = "relational_integrity_v1"

func NewSchemaMaintenanceRepository(db *sql.DB) *SchemaMaintenanceRepository {
	return &SchemaMaintenanceRepository{DB: db}
}

func (r *SchemaMaintenanceRepository) EnsureRelationalIntegrity() error {
	if err := r.ensureMaintenanceStateTable(); err != nil {
		return err
	}

	applied, err := r.isMaintenanceStepApplied(relationalIntegrityMigrationKey)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	satisfied, err := r.isRelationalIntegritySatisfied()
	if err != nil {
		return err
	}
	if satisfied {
		return r.markMaintenanceStepApplied(relationalIntegrityMigrationKey)
	}

	if err := r.runRelationalIntegrityMigration(); err != nil {
		return err
	}

	return r.markMaintenanceStepApplied(relationalIntegrityMigrationKey)
}

func (r *SchemaMaintenanceRepository) runRelationalIntegrityMigration() error {
	companyRepo := NewCompanyContactRepository(r.DB)
	if err := companyRepo.SyncFromExistingData(DefaultCompanyContactEmail); err != nil {
		return fmt.Errorf("error syncing company contacts before relational migration: %w", err)
	}
	if err := companyRepo.BackfillOrderReferences(); err != nil {
		return fmt.Errorf("error backfilling order company references before relational migration: %w", err)
	}

	if err := r.backfillHoaDonCompanyReferences(); err != nil {
		return err
	}
	if err := r.backfillInvoiceCompanyReferences(); err != nil {
		return err
	}
	if err := r.ensureColumnDefinitions(); err != nil {
		return err
	}
	if err := r.ensureIndexes(); err != nil {
		return err
	}
	if err := r.cleanupNullableOrphans(); err != nil {
		return err
	}
	if err := r.ensureRequiredReferences(); err != nil {
		return err
	}
	if err := r.ensureUniqueIndexes(); err != nil {
		return err
	}
	if err := r.ensureForeignKeys(); err != nil {
		return err
	}
	if err := r.dropUnusedTables(); err != nil {
		return err
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureMaintenanceStateTable() error {
	statement := `
		CREATE TABLE IF NOT EXISTS schema_maintenance_state (
			step_name VARCHAR(100) NOT NULL,
			completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (step_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error ensuring schema maintenance state table: %w", err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) isMaintenanceStepApplied(stepName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM schema_maintenance_state
		WHERE step_name = ?
	`, stepName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking schema maintenance step %s: %w", stepName, err)
	}

	return count > 0, nil
}

func (r *SchemaMaintenanceRepository) markMaintenanceStepApplied(stepName string) error {
	if _, err := r.DB.Exec(`
		INSERT INTO schema_maintenance_state (step_name, completed_at)
		VALUES (?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE completed_at = VALUES(completed_at)
	`, stepName); err != nil {
		return fmt.Errorf("error marking schema maintenance step %s as applied: %w", stepName, err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) isRelationalIntegritySatisfied() (bool, error) {
	exists, err := r.tableExists("company_contacts")
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	giaThauExists, err := r.tableExists("gia_thau")
	if err != nil {
		return false, err
	}
	if giaThauExists {
		return false, nil
	}

	requiredUniqueIndexes := []struct {
		tableName string
		indexName string
	}{
		{tableName: "hoa_don", indexName: "uq_hoa_don_invoice_line"},
		{tableName: "so_sanh_vat_tu", indexName: "uq_so_sanh_vat_tu_ma_thu_vien"},
	}

	for _, index := range requiredUniqueIndexes {
		exists, err := r.indexExists(index.tableName, index.indexName)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	requiredForeignKeys := []struct {
		tableName      string
		constraintName string
	}{
		{tableName: "pending_orders", constraintName: "fk_pending_orders_company_contact"},
		{tableName: "pending_orders", constraintName: "fk_pending_orders_approver_user"},
		{tableName: "pending_orders", constraintName: "fk_pending_orders_creator_user"},
		{tableName: "order_history", constraintName: "fk_order_history_company_contact"},
		{tableName: "order_history", constraintName: "fk_order_history_approver_user"},
		{tableName: "order_history", constraintName: "fk_order_history_creator_user"},
		{tableName: "order_history", constraintName: "fk_order_history_placed_by_user"},
		{tableName: "hoa_don", constraintName: "fk_hoa_don_company_contact"},
		{tableName: "order_group_reads", constraintName: "fk_order_group_reads_user"},
		{tableName: "supplier_alert_reads", constraintName: "fk_supplier_alert_reads_user"},
		{tableName: "forecast_approvals", constraintName: "fk_forecast_approvals_reviewer_user"},
		{tableName: "forecast_change_history", constraintName: "fk_forecast_change_history_actor_user"},
		{tableName: "order_invoice_reconciliation", constraintName: "fk_oir_order_history"},
		{tableName: "order_invoice_reconciliation", constraintName: "fk_oir_company_contact"},
		{tableName: "order_invoice_reconciliation", constraintName: "fk_oir_invoice_company_contact"},
		{tableName: "order_invoice_reconciliation", constraintName: "fk_oir_matched_by_user"},
	}

	for _, foreignKey := range requiredForeignKeys {
		exists, err := r.foreignKeyExists(foreignKey.tableName, foreignKey.constraintName)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	return true, nil
}

func (r *SchemaMaintenanceRepository) backfillHoaDonCompanyReferences() error {
	statements := []string{
		`
		UPDATE hoa_don h
		JOIN (
			SELECT MIN(id) AS id, tax_id
			FROM company_contacts
			WHERE TRIM(COALESCE(tax_id, '')) <> ''
			GROUP BY tax_id
		) cc
			ON CONVERT(TRIM(COALESCE(h.ma_so_thue_nguoi_ban, '')) USING utf8mb4) COLLATE utf8mb4_unicode_ci = cc.tax_id
		SET h.company_contact_id = cc.id
		WHERE h.company_contact_id IS NULL
		  AND TRIM(COALESCE(h.ma_so_thue_nguoi_ban, '')) <> ''
		`,
		`
		UPDATE hoa_don h
		JOIN (
			SELECT MIN(id) AS id, company_name
			FROM company_contacts
			WHERE TRIM(COALESCE(company_name, '')) <> ''
			GROUP BY company_name
		) cc
			ON CONVERT(TRIM(COALESCE(h.cong_ty, '')) USING utf8mb4) COLLATE utf8mb4_unicode_ci = cc.company_name
		SET h.company_contact_id = cc.id
		WHERE h.company_contact_id IS NULL
		  AND TRIM(COALESCE(h.cong_ty, '')) <> ''
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error backfilling hoa_don company references: %w", err)
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) backfillInvoiceCompanyReferences() error {
	statements := []string{
		`
		UPDATE order_invoice_reconciliation r
		JOIN (
			SELECT MIN(id) AS id, company_name
			FROM company_contacts
			WHERE TRIM(COALESCE(company_name, '')) <> ''
			GROUP BY company_name
		) cc
			ON CONVERT(TRIM(COALESCE(r.invoice_company_name, '')) USING utf8mb4) COLLATE utf8mb4_unicode_ci = cc.company_name
		SET r.invoice_company_contact_id = cc.id
		WHERE r.invoice_company_contact_id IS NULL
		  AND TRIM(COALESCE(r.invoice_company_name, '')) <> ''
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error backfilling invoice company references: %w", err)
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureColumnDefinitions() error {
	type columnChange struct {
		tableName      string
		columnName     string
		expectedType   string
		expectedNull   bool
		alterStatement string
	}

	changes := []columnChange{
		{
			tableName:      "hoa_don",
			columnName:     "company_contact_id",
			expectedType:   "bigint",
			expectedNull:   true,
			alterStatement: "ALTER TABLE hoa_don MODIFY COLUMN company_contact_id BIGINT NULL",
		},
		{
			tableName:      "order_group_reads",
			columnName:     "user_id",
			expectedType:   "bigint unsigned",
			expectedNull:   false,
			alterStatement: "ALTER TABLE order_group_reads MODIFY COLUMN user_id BIGINT UNSIGNED NOT NULL",
		},
		{
			tableName:      "supplier_alert_reads",
			columnName:     "user_id",
			expectedType:   "bigint unsigned",
			expectedNull:   false,
			alterStatement: "ALTER TABLE supplier_alert_reads MODIFY COLUMN user_id BIGINT UNSIGNED NOT NULL",
		},
		{
			tableName:      "pending_orders",
			columnName:     "nguoi_phe_duyet_id",
			expectedType:   "bigint unsigned",
			expectedNull:   true,
			alterStatement: "ALTER TABLE pending_orders MODIFY COLUMN nguoi_phe_duyet_id BIGINT UNSIGNED NULL",
		},
		{
			tableName:      "pending_orders",
			columnName:     "nguoi_tao_don_id",
			expectedType:   "bigint unsigned",
			expectedNull:   true,
			alterStatement: "ALTER TABLE pending_orders MODIFY COLUMN nguoi_tao_don_id BIGINT UNSIGNED NULL",
		},
		{
			tableName:      "order_history",
			columnName:     "nguoi_phe_duyet_id",
			expectedType:   "bigint unsigned",
			expectedNull:   true,
			alterStatement: "ALTER TABLE order_history MODIFY COLUMN nguoi_phe_duyet_id BIGINT UNSIGNED NULL",
		},
		{
			tableName:      "order_history",
			columnName:     "nguoi_tao_don_id",
			expectedType:   "bigint unsigned",
			expectedNull:   true,
			alterStatement: "ALTER TABLE order_history MODIFY COLUMN nguoi_tao_don_id BIGINT UNSIGNED NULL",
		},
		{
			tableName:      "order_history",
			columnName:     "nguoi_dat_hang_id",
			expectedType:   "bigint unsigned",
			expectedNull:   false,
			alterStatement: "ALTER TABLE order_history MODIFY COLUMN nguoi_dat_hang_id BIGINT UNSIGNED NOT NULL",
		},
		{
			tableName:      "forecast_approvals",
			columnName:     "nguoi_duyet_id",
			expectedType:   "bigint unsigned",
			expectedNull:   false,
			alterStatement: "ALTER TABLE forecast_approvals MODIFY COLUMN nguoi_duyet_id BIGINT UNSIGNED NOT NULL",
		},
		{
			tableName:      "forecast_change_history",
			columnName:     "nguoi_thuc_hien_id",
			expectedType:   "bigint unsigned",
			expectedNull:   false,
			alterStatement: "ALTER TABLE forecast_change_history MODIFY COLUMN nguoi_thuc_hien_id BIGINT UNSIGNED NOT NULL",
		},
		{
			tableName:      "order_invoice_reconciliation",
			columnName:     "matched_by_user_id",
			expectedType:   "bigint unsigned",
			expectedNull:   true,
			alterStatement: "ALTER TABLE order_invoice_reconciliation MODIFY COLUMN matched_by_user_id BIGINT UNSIGNED NULL",
		},
	}

	for _, change := range changes {
		if err := r.ensureColumnDefinition(change.tableName, change.columnName, change.expectedType, change.expectedNull, change.alterStatement); err != nil {
			return err
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureIndexes() error {
	type indexChange struct {
		tableName      string
		indexName      string
		alterStatement string
	}

	indexes := []indexChange{
		{
			tableName:      "hoa_don",
			indexName:      "idx_hoa_don_company_contact",
			alterStatement: "ALTER TABLE hoa_don ADD INDEX idx_hoa_don_company_contact (company_contact_id)",
		},
		{
			tableName:      "pending_orders",
			indexName:      "idx_pending_orders_approver_user",
			alterStatement: "ALTER TABLE pending_orders ADD INDEX idx_pending_orders_approver_user (nguoi_phe_duyet_id)",
		},
		{
			tableName:      "pending_orders",
			indexName:      "idx_pending_orders_creator_user",
			alterStatement: "ALTER TABLE pending_orders ADD INDEX idx_pending_orders_creator_user (nguoi_tao_don_id)",
		},
		{
			tableName:      "order_history",
			indexName:      "idx_order_history_approver_user",
			alterStatement: "ALTER TABLE order_history ADD INDEX idx_order_history_approver_user (nguoi_phe_duyet_id)",
		},
		{
			tableName:      "order_history",
			indexName:      "idx_order_history_creator_user",
			alterStatement: "ALTER TABLE order_history ADD INDEX idx_order_history_creator_user (nguoi_tao_don_id)",
		},
		{
			tableName:      "order_history",
			indexName:      "idx_order_history_placed_by_user",
			alterStatement: "ALTER TABLE order_history ADD INDEX idx_order_history_placed_by_user (nguoi_dat_hang_id)",
		},
		{
			tableName:      "forecast_approvals",
			indexName:      "idx_forecast_approvals_reviewer_user",
			alterStatement: "ALTER TABLE forecast_approvals ADD INDEX idx_forecast_approvals_reviewer_user (nguoi_duyet_id)",
		},
		{
			tableName:      "forecast_change_history",
			indexName:      "idx_forecast_change_history_actor_user",
			alterStatement: "ALTER TABLE forecast_change_history ADD INDEX idx_forecast_change_history_actor_user (nguoi_thuc_hien_id)",
		},
		{
			tableName:      "order_invoice_reconciliation",
			indexName:      "idx_oir_company_contact",
			alterStatement: "ALTER TABLE order_invoice_reconciliation ADD INDEX idx_oir_company_contact (company_contact_id)",
		},
		{
			tableName:      "order_invoice_reconciliation",
			indexName:      "idx_oir_invoice_company_contact",
			alterStatement: "ALTER TABLE order_invoice_reconciliation ADD INDEX idx_oir_invoice_company_contact (invoice_company_contact_id)",
		},
		{
			tableName:      "order_invoice_reconciliation",
			indexName:      "idx_oir_matched_by_user",
			alterStatement: "ALTER TABLE order_invoice_reconciliation ADD INDEX idx_oir_matched_by_user (matched_by_user_id)",
		},
	}

	for _, index := range indexes {
		if err := r.ensureIndexExists(index.tableName, index.indexName, index.alterStatement); err != nil {
			return err
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) cleanupNullableOrphans() error {
	statements := []string{
		`
		UPDATE pending_orders p
		LEFT JOIN company_contacts c ON c.id = p.company_contact_id
		SET p.company_contact_id = NULL
		WHERE p.company_contact_id IS NOT NULL AND c.id IS NULL
		`,
		`
		UPDATE order_history o
		LEFT JOIN company_contacts c ON c.id = o.company_contact_id
		SET o.company_contact_id = NULL
		WHERE o.company_contact_id IS NOT NULL AND c.id IS NULL
		`,
		`
		UPDATE hoa_don h
		LEFT JOIN company_contacts c ON c.id = h.company_contact_id
		SET h.company_contact_id = NULL
		WHERE h.company_contact_id IS NOT NULL AND c.id IS NULL
		`,
		`
		UPDATE order_invoice_reconciliation r
		LEFT JOIN company_contacts c ON c.id = r.company_contact_id
		SET r.company_contact_id = NULL
		WHERE r.company_contact_id IS NOT NULL AND c.id IS NULL
		`,
		`
		UPDATE order_invoice_reconciliation r
		LEFT JOIN company_contacts c ON c.id = r.invoice_company_contact_id
		SET r.invoice_company_contact_id = NULL
		WHERE r.invoice_company_contact_id IS NOT NULL AND c.id IS NULL
		`,
		`
		UPDATE pending_orders p
		LEFT JOIN users u ON u.id = p.nguoi_phe_duyet_id
		SET p.nguoi_phe_duyet_id = NULL
		WHERE p.nguoi_phe_duyet_id IS NOT NULL AND u.id IS NULL
		`,
		`
		UPDATE pending_orders p
		LEFT JOIN users u ON u.id = p.nguoi_tao_don_id
		SET p.nguoi_tao_don_id = NULL
		WHERE p.nguoi_tao_don_id IS NOT NULL AND u.id IS NULL
		`,
		`
		UPDATE order_history o
		LEFT JOIN users u ON u.id = o.nguoi_phe_duyet_id
		SET o.nguoi_phe_duyet_id = NULL
		WHERE o.nguoi_phe_duyet_id IS NOT NULL AND u.id IS NULL
		`,
		`
		UPDATE order_history o
		LEFT JOIN users u ON u.id = o.nguoi_tao_don_id
		SET o.nguoi_tao_don_id = NULL
		WHERE o.nguoi_tao_don_id IS NOT NULL AND u.id IS NULL
		`,
		`
		UPDATE order_invoice_reconciliation r
		LEFT JOIN users u ON u.id = r.matched_by_user_id
		SET r.matched_by_user_id = NULL
		WHERE r.matched_by_user_id IS NOT NULL AND u.id IS NULL
		`,
		`
		DELETE g
		FROM order_group_reads g
		LEFT JOIN users u ON u.id = g.user_id
		WHERE u.id IS NULL
		`,
		`
		DELETE s
		FROM supplier_alert_reads s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE u.id IS NULL
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error cleaning nullable orphan references: %w", err)
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureRequiredReferences() error {
	type validation struct {
		name      string
		statement string
	}

	validations := []validation{
		{
			name: "order_history.nguoi_dat_hang_id",
			statement: `
				SELECT COUNT(*)
				FROM order_history o
				LEFT JOIN users u ON u.id = o.nguoi_dat_hang_id
				WHERE u.id IS NULL
			`,
		},
		{
			name: "forecast_approvals.nguoi_duyet_id",
			statement: `
				SELECT COUNT(*)
				FROM forecast_approvals f
				LEFT JOIN users u ON u.id = f.nguoi_duyet_id
				WHERE u.id IS NULL
			`,
		},
		{
			name: "forecast_change_history.nguoi_thuc_hien_id",
			statement: `
				SELECT COUNT(*)
				FROM forecast_change_history f
				LEFT JOIN users u ON u.id = f.nguoi_thuc_hien_id
				WHERE u.id IS NULL
			`,
		},
		{
			name: "order_invoice_reconciliation.order_history_id",
			statement: `
				SELECT COUNT(*)
				FROM order_invoice_reconciliation r
				LEFT JOIN order_history o ON o.id = r.order_history_id
				WHERE o.id IS NULL
			`,
		},
	}

	for _, validation := range validations {
		var count int
		if err := r.DB.QueryRow(validation.statement).Scan(&count); err != nil {
			return fmt.Errorf("error validating %s: %w", validation.name, err)
		}
		if count > 0 {
			return fmt.Errorf("%s still has %d orphaned rows", validation.name, count)
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureUniqueIndexes() error {
	type uniqueIndex struct {
		tableName  string
		indexName  string
		columns    string
		duplicates string
	}

	indexes := []uniqueIndex{
		{
			tableName:  "hoa_don",
			indexName:  "uq_hoa_don_invoice_line",
			columns:    "id_hoa_don, stt_dong_hang",
			duplicates: "SELECT COUNT(*) FROM (SELECT id_hoa_don, stt_dong_hang FROM hoa_don WHERE id_hoa_don IS NOT NULL AND stt_dong_hang IS NOT NULL GROUP BY id_hoa_don, stt_dong_hang HAVING COUNT(*) > 1) dup",
		},
		{
			tableName:  "so_sanh_vat_tu",
			indexName:  "uq_so_sanh_vat_tu_ma_thu_vien",
			columns:    "ma_thu_vien",
			duplicates: "SELECT COUNT(*) FROM (SELECT ma_thu_vien FROM so_sanh_vat_tu WHERE ma_thu_vien IS NOT NULL AND TRIM(ma_thu_vien) <> '' GROUP BY ma_thu_vien HAVING COUNT(*) > 1) dup",
		},
	}

	for _, index := range indexes {
		if err := r.ensureUniqueIndex(index.tableName, index.indexName, index.columns, index.duplicates); err != nil {
			return err
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureForeignKeys() error {
	type foreignKey struct {
		tableName      string
		constraintName string
		statement      string
	}

	foreignKeys := []foreignKey{
		{
			tableName:      "pending_orders",
			constraintName: "fk_pending_orders_company_contact",
			statement:      "ALTER TABLE pending_orders ADD CONSTRAINT fk_pending_orders_company_contact FOREIGN KEY (company_contact_id) REFERENCES company_contacts (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "pending_orders",
			constraintName: "fk_pending_orders_approver_user",
			statement:      "ALTER TABLE pending_orders ADD CONSTRAINT fk_pending_orders_approver_user FOREIGN KEY (nguoi_phe_duyet_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "pending_orders",
			constraintName: "fk_pending_orders_creator_user",
			statement:      "ALTER TABLE pending_orders ADD CONSTRAINT fk_pending_orders_creator_user FOREIGN KEY (nguoi_tao_don_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_history",
			constraintName: "fk_order_history_company_contact",
			statement:      "ALTER TABLE order_history ADD CONSTRAINT fk_order_history_company_contact FOREIGN KEY (company_contact_id) REFERENCES company_contacts (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_history",
			constraintName: "fk_order_history_approver_user",
			statement:      "ALTER TABLE order_history ADD CONSTRAINT fk_order_history_approver_user FOREIGN KEY (nguoi_phe_duyet_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_history",
			constraintName: "fk_order_history_creator_user",
			statement:      "ALTER TABLE order_history ADD CONSTRAINT fk_order_history_creator_user FOREIGN KEY (nguoi_tao_don_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_history",
			constraintName: "fk_order_history_placed_by_user",
			statement:      "ALTER TABLE order_history ADD CONSTRAINT fk_order_history_placed_by_user FOREIGN KEY (nguoi_dat_hang_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT",
		},
		{
			tableName:      "hoa_don",
			constraintName: "fk_hoa_don_company_contact",
			statement:      "ALTER TABLE hoa_don ADD CONSTRAINT fk_hoa_don_company_contact FOREIGN KEY (company_contact_id) REFERENCES company_contacts (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_group_reads",
			constraintName: "fk_order_group_reads_user",
			statement:      "ALTER TABLE order_group_reads ADD CONSTRAINT fk_order_group_reads_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE",
		},
		{
			tableName:      "supplier_alert_reads",
			constraintName: "fk_supplier_alert_reads_user",
			statement:      "ALTER TABLE supplier_alert_reads ADD CONSTRAINT fk_supplier_alert_reads_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE",
		},
		{
			tableName:      "forecast_approvals",
			constraintName: "fk_forecast_approvals_reviewer_user",
			statement:      "ALTER TABLE forecast_approvals ADD CONSTRAINT fk_forecast_approvals_reviewer_user FOREIGN KEY (nguoi_duyet_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT",
		},
		{
			tableName:      "forecast_change_history",
			constraintName: "fk_forecast_change_history_actor_user",
			statement:      "ALTER TABLE forecast_change_history ADD CONSTRAINT fk_forecast_change_history_actor_user FOREIGN KEY (nguoi_thuc_hien_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE RESTRICT",
		},
		{
			tableName:      "order_invoice_reconciliation",
			constraintName: "fk_oir_order_history",
			statement:      "ALTER TABLE order_invoice_reconciliation ADD CONSTRAINT fk_oir_order_history FOREIGN KEY (order_history_id) REFERENCES order_history (id) ON UPDATE CASCADE ON DELETE RESTRICT",
		},
		{
			tableName:      "order_invoice_reconciliation",
			constraintName: "fk_oir_company_contact",
			statement:      "ALTER TABLE order_invoice_reconciliation ADD CONSTRAINT fk_oir_company_contact FOREIGN KEY (company_contact_id) REFERENCES company_contacts (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_invoice_reconciliation",
			constraintName: "fk_oir_invoice_company_contact",
			statement:      "ALTER TABLE order_invoice_reconciliation ADD CONSTRAINT fk_oir_invoice_company_contact FOREIGN KEY (invoice_company_contact_id) REFERENCES company_contacts (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
		{
			tableName:      "order_invoice_reconciliation",
			constraintName: "fk_oir_matched_by_user",
			statement:      "ALTER TABLE order_invoice_reconciliation ADD CONSTRAINT fk_oir_matched_by_user FOREIGN KEY (matched_by_user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE SET NULL",
		},
	}

	for _, foreignKey := range foreignKeys {
		if err := r.ensureForeignKey(foreignKey.tableName, foreignKey.constraintName, foreignKey.statement); err != nil {
			return err
		}
	}

	return nil
}

func (r *SchemaMaintenanceRepository) dropUnusedTables() error {
	exists, err := r.tableExists("gia_thau")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if _, err := r.DB.Exec("DROP TABLE IF EXISTS gia_thau"); err != nil {
		return fmt.Errorf("error dropping unused table gia_thau: %w", err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) tableExists(tableName string) (bool, error) {
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

func (r *SchemaMaintenanceRepository) ensureColumnDefinition(tableName, columnName, expectedType string, expectedNull bool, alterStatement string) error {
	var columnType string
	var isNullable string
	if err := r.DB.QueryRow(`
		SELECT COLUMN_TYPE, IS_NULLABLE
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, tableName, columnName).Scan(&columnType, &isNullable); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("missing required column %s.%s", tableName, columnName)
		}
		return fmt.Errorf("error checking column definition for %s.%s: %w", tableName, columnName, err)
	}

	if strings.EqualFold(columnType, expectedType) && ((expectedNull && isNullable == "YES") || (!expectedNull && isNullable == "NO")) {
		return nil
	}

	if _, err := r.DB.Exec(alterStatement); err != nil {
		return fmt.Errorf("error altering %s.%s: %w", tableName, columnName, err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureIndexExists(tableName, indexName, alterStatement string) error {
	exists, err := r.indexExists(tableName, indexName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if _, err := r.DB.Exec(alterStatement); err != nil {
		return fmt.Errorf("error creating index %s on %s: %w", indexName, tableName, err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureUniqueIndex(tableName, indexName, columns, duplicateQuery string) error {
	exists, err := r.indexExists(tableName, indexName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	var duplicateCount int
	if err := r.DB.QueryRow(duplicateQuery).Scan(&duplicateCount); err != nil {
		return fmt.Errorf("error validating unique index %s on %s: %w", indexName, tableName, err)
	}
	if duplicateCount > 0 {
		return fmt.Errorf("cannot create unique index %s on %s because %d duplicate groups still exist", indexName, tableName, duplicateCount)
	}

	statement := fmt.Sprintf("ALTER TABLE %s ADD UNIQUE INDEX %s (%s)", tableName, indexName, columns)
	if _, err := r.DB.Exec(statement); err != nil {
		return fmt.Errorf("error creating unique index %s on %s: %w", indexName, tableName, err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) ensureForeignKey(tableName, constraintName, alterStatement string) error {
	exists, err := r.foreignKeyExists(tableName, constraintName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if _, err := r.DB.Exec(alterStatement); err != nil {
		return fmt.Errorf("error creating foreign key %s on %s: %w", constraintName, tableName, err)
	}

	return nil
}

func (r *SchemaMaintenanceRepository) indexExists(tableName, indexName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
	`, tableName, indexName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking index %s on %s: %w", indexName, tableName, err)
	}

	return count > 0, nil
}

func (r *SchemaMaintenanceRepository) foreignKeyExists(tableName, constraintName string) (bool, error) {
	var count int
	if err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.table_constraints
		WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'
	`, tableName, constraintName).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking foreign key %s on %s: %w", constraintName, tableName, err)
	}

	return count > 0, nil
}
