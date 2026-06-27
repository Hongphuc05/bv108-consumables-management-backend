package models

import (
	"database/sql"
	"fmt"
	"strings"
)

const supplyVisibilityScopeGlobal = "global"

type SupplyTaskRepository struct {
	DB *sql.DB
}

type SupplyTaskAssignedSupply struct {
	IDX1 int    `json:"idx1"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SupplyTaskExportRow struct {
	IDX1              int
	ProductID         sql.NullInt32
	GroupName         sql.NullString
	ID                sql.NullString
	IDX2              sql.NullString
	MaHieu            sql.NullString
	TypeName          sql.NullString
	Name              sql.NullString
	Unit              sql.NullString
	QuyCachDongGoi    sql.NullString
	QuyCachGiaoHang   sql.NullString
	QuyCachToiThieu   sql.NullString
	ThongTinThau      sql.NullString
	TongThau          sql.NullString
	HangSX            sql.NullString
	NuocSX            sql.NullString
	NhaCungCap        sql.NullString
	Price             sql.NullFloat64
	TonDauKy          sql.NullInt32
	NhapTrongKy       sql.NullInt32
	XuatTrongKy       sql.NullInt32
	TongNhap          sql.NullInt32
	TonKhoMin         sql.NullInt32
	AssignedToUserID  sql.NullInt64
}

type SupplyTaskImportAssignment struct {
	SupplyIDX1 int
	UserID     int64
	Assigned   bool
}

func NewSupplyTaskRepository(db *sql.DB) *SupplyTaskRepository {
	return &SupplyTaskRepository{DB: db}
}

func (r *SupplyTaskRepository) EnsureSchema() error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS supply_visibility_settings (
			scope_key VARCHAR(64) NOT NULL,
			hide_for_other_roles TINYINT(1) NOT NULL DEFAULT 0,
			updated_by_user_id BIGINT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (scope_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
		`
		CREATE TABLE IF NOT EXISTS supply_user_assignments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			supply_idx1 INT NOT NULL,
			assigned_by_user_id BIGINT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_supply_user_assignments_user_supply (user_id, supply_idx1),
			KEY idx_supply_user_assignments_user (user_id),
			KEY idx_supply_user_assignments_supply (supply_idx1)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error ensuring supply task schema: %w", err)
		}
	}

	if _, err := r.DB.Exec(`
		INSERT INTO supply_visibility_settings (scope_key, hide_for_other_roles)
		VALUES (?, 0)
		ON DUPLICATE KEY UPDATE scope_key = VALUES(scope_key)
	`, supplyVisibilityScopeGlobal); err != nil {
		return fmt.Errorf("error seeding supply visibility settings: %w", err)
	}

	return nil
}

func (r *SupplyTaskRepository) IsHideForOtherRolesEnabled() (bool, error) {
	var hide int
	err := r.DB.QueryRow(`
		SELECT hide_for_other_roles
		FROM supply_visibility_settings
		WHERE scope_key = ?
	`, supplyVisibilityScopeGlobal).Scan(&hide)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error reading visibility setting: %w", err)
	}

	return hide == 1, nil
}

func (r *SupplyTaskRepository) SetHideForOtherRolesEnabled(enabled bool, updatedByUserID int64) error {
	hideValue := 0
	if enabled {
		hideValue = 1
	}

	if _, err := r.DB.Exec(`
		INSERT INTO supply_visibility_settings (scope_key, hide_for_other_roles, updated_by_user_id)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			hide_for_other_roles = VALUES(hide_for_other_roles),
			updated_by_user_id = VALUES(updated_by_user_id)
	`, supplyVisibilityScopeGlobal, hideValue, updatedByUserID); err != nil {
		return fmt.Errorf("error updating visibility setting: %w", err)
	}

	return nil
}

func (r *SupplyTaskRepository) GetAssignedSupplyIDX1ByUserID(userID int64) ([]int, error) {
	rows, err := r.DB.Query(`
		SELECT supply_idx1
		FROM supply_user_assignments
		WHERE user_id = ?
		ORDER BY supply_idx1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying assigned supplies: %w", err)
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var idx1 int
		if err := rows.Scan(&idx1); err != nil {
			return nil, fmt.Errorf("error scanning assigned supply idx1: %w", err)
		}
		ids = append(ids, idx1)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assigned supplies: %w", err)
	}

	return ids, nil
}

func (r *SupplyTaskRepository) GetAssignedSupplyDetailsByUserID(userID int64) ([]SupplyTaskAssignedSupply, error) {
	rows, err := r.DB.Query(`
		SELECT s.IDX1, COALESCE(s.ID, ''), COALESCE(s.NAME, '')
		FROM supply_user_assignments sua
		INNER JOIN supplies s ON s.IDX1 = sua.supply_idx1
		WHERE sua.user_id = ?
		ORDER BY s.IDX1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying assigned supply details: %w", err)
	}
	defer rows.Close()

	items := make([]SupplyTaskAssignedSupply, 0)
	for rows.Next() {
		var item SupplyTaskAssignedSupply
		if err := rows.Scan(&item.IDX1, &item.Code, &item.Name); err != nil {
			return nil, fmt.Errorf("error scanning assigned supply detail: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assigned supply details: %w", err)
	}

	return items, nil
}

func (r *SupplyTaskRepository) ReplaceAssignmentsForUser(userID int64, supplyIDX1List []int, assignedByUserID int64) error {
	oldIDs, err := r.GetAssignedSupplyIDX1ByUserID(userID)
	if err != nil {
		return fmt.Errorf("error getting current assignments: %w", err)
	}

	oldMap := make(map[int]bool, len(oldIDs))
	for _, id := range oldIDs {
		oldMap[id] = true
	}

	newMap := make(map[int]bool, len(supplyIDX1List))
	for _, id := range supplyIDX1List {
		newMap[id] = true
	}

	var toAdd []int
	for _, id := range supplyIDX1List {
		if !oldMap[id] {
			toAdd = append(toAdd, id)
		}
	}

	var toDelete []int
	for _, id := range oldIDs {
		if !newMap[id] {
			toDelete = append(toDelete, id)
		}
	}

	if len(supplyIDX1List) == 0 && len(toDelete) == 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error beginning assignment transaction: %w", err)
	}
	defer tx.Rollback()

	const batchSize = 500

	if len(toDelete) > 0 {
		for i := 0; i < len(toDelete); i += batchSize {
			end := i + batchSize
			if end > len(toDelete) {
				end = len(toDelete)
			}
			batch := toDelete[i:end]

			placeholders := strings.TrimRight(strings.Repeat("?,", len(batch)), ",")
			query := fmt.Sprintf("DELETE FROM supply_user_assignments WHERE user_id = ? AND supply_idx1 IN (%s)", placeholders)

			args := make([]interface{}, 0, 1+len(batch))
			args = append(args, userID)
			for _, id := range batch {
				args = append(args, id)
			}

			if _, err := tx.Exec(query, args...); err != nil {
				return fmt.Errorf("error deleting batch: %w", err)
			}
		}
	}

	if len(supplyIDX1List) > 0 {
		for i := 0; i < len(supplyIDX1List); i += batchSize {
			end := i + batchSize
			if end > len(supplyIDX1List) {
				end = len(supplyIDX1List)
			}
			batch := supplyIDX1List[i:end]

			otherUserArgs := make([]interface{}, 0, 1+len(batch))
			otherUserArgs = append(otherUserArgs, userID)
			for _, id := range batch {
				otherUserArgs = append(otherUserArgs, id)
			}

			if _, err := tx.Exec(
				fmt.Sprintf(
					"DELETE FROM supply_user_assignments WHERE user_id <> ? AND supply_idx1 IN (%s)",
					strings.TrimRight(strings.Repeat("?,", len(batch)), ","),
				),
				otherUserArgs...,
			); err != nil {
				return fmt.Errorf("error clearing existing owners before insert: %w", err)
			}
		}
	}

	if len(toAdd) > 0 {
		for i := 0; i < len(toAdd); i += batchSize {
			end := i + batchSize
			if end > len(toAdd) {
				end = len(toAdd)
			}
			batch := toAdd[i:end]

			valuePlaceholders := make([]string, len(batch))
			args := make([]interface{}, 0, len(batch)*3)

			for idx, id := range batch {
				valuePlaceholders[idx] = "(?, ?, ?)"
				args = append(args, userID, id, assignedByUserID)
			}

			query := fmt.Sprintf(
				"INSERT INTO supply_user_assignments (user_id, supply_idx1, assigned_by_user_id) VALUES %s",
				strings.Join(valuePlaceholders, ","),
			)

			if _, err := tx.Exec(query, args...); err != nil {
				return fmt.Errorf("error inserting batch: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing assignment transaction: %w", err)
	}

	return nil
}

func (r *SupplyTaskRepository) GetAssignedCountsByUserIDs(userIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int)
	if len(userIDs) == 0 {
		return counts, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(userIDs)), ",")
	args := make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		args = append(args, userID)
	}

	query := fmt.Sprintf(`
		SELECT user_id, COUNT(*) AS total
		FROM supply_user_assignments
		WHERE user_id IN (%s)
		GROUP BY user_id
	`, placeholders)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying assignment counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var total int
		if err := rows.Scan(&userID, &total); err != nil {
			return nil, fmt.Errorf("error scanning assignment count: %w", err)
		}
		counts[userID] = total
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignment counts: %w", err)
	}

	return counts, nil
}

func (r *SupplyTaskRepository) ListExportRows() ([]SupplyTaskExportRow, error) {
	rows, err := r.DB.Query(`
		SELECT
			s.IDX1,
			s.PRODUCTID,
			s.GROUPNAME,
			s.ID,
			s.IDX2,
			s.MA_HIEU,
			s.TYPENAME,
			s.NAME,
			s.UNIT,
			s.QUY_CACH_DONG_GOI,
			s.QUY_CACH_GIAO_HANG,
			s.QUY_CACH_TOI_THIEU,
			s.THONG_TIN_THAU,
			s.TONGTHAU,
			s.HANGSX,
			s.NUOC_SX,
			s.NHA_CUNG_CAP,
			s.PRICE,
			s.TONDAUKY,
			s.NHAPTRONGKY,
			s.XUATTRONGKY,
			s.TONGNHAP,
			s.TON_KHO_MIN,
			owner.user_id
		FROM supplies s
		LEFT JOIN (
			SELECT supply_idx1, MIN(user_id) AS user_id
			FROM supply_user_assignments
			GROUP BY supply_idx1
		) owner ON owner.supply_idx1 = s.IDX1
		ORDER BY s.IDX1
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying export rows: %w", err)
	}
	defer rows.Close()

	items := make([]SupplyTaskExportRow, 0)
	for rows.Next() {
		var item SupplyTaskExportRow
		if err := rows.Scan(
			&item.IDX1,
			&item.ProductID,
			&item.GroupName,
			&item.ID,
			&item.IDX2,
			&item.MaHieu,
			&item.TypeName,
			&item.Name,
			&item.Unit,
			&item.QuyCachDongGoi,
			&item.QuyCachGiaoHang,
			&item.QuyCachToiThieu,
			&item.ThongTinThau,
			&item.TongThau,
			&item.HangSX,
			&item.NuocSX,
			&item.NhaCungCap,
			&item.Price,
			&item.TonDauKy,
			&item.NhapTrongKy,
			&item.XuatTrongKy,
			&item.TongNhap,
			&item.TonKhoMin,
			&item.AssignedToUserID,
		); err != nil {
			return nil, fmt.Errorf("error scanning export row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating export rows: %w", err)
	}

	return items, nil
}

func (r *SupplyTaskRepository) GetExistingSupplyIDX1Set(idx1List []int) (map[int]struct{}, error) {
	result := make(map[int]struct{})
	if len(idx1List) == 0 {
		return result, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(idx1List)), ",")
	args := make([]interface{}, 0, len(idx1List))
	for _, idx1 := range idx1List {
		args = append(args, idx1)
	}

	rows, err := r.DB.Query(
		fmt.Sprintf("SELECT IDX1 FROM supplies WHERE IDX1 IN (%s)", placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying existing supply ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var idx1 int
		if err := rows.Scan(&idx1); err != nil {
			return nil, fmt.Errorf("error scanning existing supply id: %w", err)
		}
		result[idx1] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating existing supply ids: %w", err)
	}

	return result, nil
}

func (r *SupplyTaskRepository) ReplaceAssignmentsBySupplyIDX1(assignments []SupplyTaskImportAssignment, assignedByUserID int64) (int, int, error) {
	if len(assignments) == 0 {
		return 0, 0, nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("error beginning bulk assignment transaction: %w", err)
	}
	defer tx.Rollback()

	supplyIDX1List := make([]int, 0, len(assignments))
	for _, item := range assignments {
		supplyIDX1List = append(supplyIDX1List, item.SupplyIDX1)
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(supplyIDX1List)), ",")
	deleteArgs := make([]interface{}, 0, len(supplyIDX1List))
	for _, idx1 := range supplyIDX1List {
		deleteArgs = append(deleteArgs, idx1)
	}

	if _, err := tx.Exec(
		fmt.Sprintf("DELETE FROM supply_user_assignments WHERE supply_idx1 IN (%s)", placeholders),
		deleteArgs...,
	); err != nil {
		return 0, 0, fmt.Errorf("error clearing existing assignments by supply: %w", err)
	}

	toAssign := make([]SupplyTaskImportAssignment, 0, len(assignments))
	clearedCount := 0
	for _, item := range assignments {
		if item.Assigned {
			toAssign = append(toAssign, item)
			continue
		}
		clearedCount++
	}

	if len(toAssign) > 0 {
		const batchSize = 500
		for i := 0; i < len(toAssign); i += batchSize {
			end := i + batchSize
			if end > len(toAssign) {
				end = len(toAssign)
			}
			batch := toAssign[i:end]

			valuePlaceholders := make([]string, len(batch))
			args := make([]interface{}, 0, len(batch)*3)
			for idx, item := range batch {
				valuePlaceholders[idx] = "(?, ?, ?)"
				args = append(args, item.UserID, item.SupplyIDX1, assignedByUserID)
			}

			if _, err := tx.Exec(
				fmt.Sprintf(
					"INSERT INTO supply_user_assignments (user_id, supply_idx1, assigned_by_user_id) VALUES %s",
					strings.Join(valuePlaceholders, ","),
				),
				args...,
			); err != nil {
				return 0, 0, fmt.Errorf("error inserting bulk assignments: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("error committing bulk assignments: %w", err)
	}

	return len(toAssign), clearedCount, nil
}
