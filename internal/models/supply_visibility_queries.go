package models

import (
	"database/sql"
	"fmt"
	"strings"
)

func buildSupplyVisibilityFilterClause(column string, visibleIDX1 []int) (string, []interface{}) {
	if visibleIDX1 == nil {
		return "", nil
	}

	if len(visibleIDX1) == 0 {
		return " AND 1 = 0", nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(visibleIDX1)), ",")
	args := make([]interface{}, 0, len(visibleIDX1))
	for _, idx1 := range visibleIDX1 {
		args = append(args, idx1)
	}

	return fmt.Sprintf(" AND %s IN (%s)", column, placeholders), args
}

func (r *SupplyRepository) CountAll() (int, error) {
	var total int
	if err := r.DB.QueryRow(`SELECT COUNT(*) FROM supplies`).Scan(&total); err != nil {
		return 0, fmt.Errorf("error counting supplies: %w", err)
	}
	return total, nil
}

func (r *SupplyRepository) GetAllVisible(page, pageSize int, visibleIDX1 []int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	var total int
	countQuery := "SELECT COUNT(*) FROM supplies WHERE 1=1" + filterClause
	if err := r.DB.QueryRow(countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE 1=1
	` + filterClause + `
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	args := append([]interface{}{}, filterArgs...)
	args = append(args, pageSize, offset)
	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

func (r *SupplyRepository) GetByIDVisible(idx1 int, visibleIDX1 []int) (*Supply, error) {
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE IDX1 = ?
	` + filterClause

	args := make([]interface{}, 0, 1+len(filterArgs))
	args = append(args, idx1)
	args = append(args, filterArgs...)

	s, err := scanSupply(r.DB.QueryRow(query, args...))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("supply not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying supply: %w", err)
	}

	return &s, nil
}

func (r *SupplyRepository) SearchByNameVisible(keyword string, page, pageSize int, visibleIDX1 []int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize
	searchPattern := "%" + keyword + "%"
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM supplies
		WHERE (NAME LIKE ? OR ID LIKE ? OR IDX2 LIKE ? OR MA_HIEU LIKE ?)
	` + filterClause
	countArgs := []interface{}{searchPattern, searchPattern, searchPattern, searchPattern}
	countArgs = append(countArgs, filterArgs...)
	if err := r.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE (NAME LIKE ? OR ID LIKE ? OR IDX2 LIKE ? OR MA_HIEU LIKE ?)
	` + filterClause + `
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	args := []interface{}{searchPattern, searchPattern, searchPattern, searchPattern}
	args = append(args, filterArgs...)
	args = append(args, pageSize, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

func (r *SupplyRepository) GetByGroupVisible(groupName string, page, pageSize int, visibleIDX1 []int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	var total int
	countQuery := "SELECT COUNT(*) FROM supplies WHERE GROUPNAME = ?" + filterClause
	countArgs := []interface{}{groupName}
	countArgs = append(countArgs, filterArgs...)
	if err := r.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error counting supplies: %w", err)
	}

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE GROUPNAME = ?
	` + filterClause + `
		ORDER BY IDX1
		LIMIT ? OFFSET ?
	`

	args := []interface{}{groupName}
	args = append(args, filterArgs...)
	args = append(args, pageSize, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

func (r *SupplyRepository) GetAllGroupsVisible(visibleIDX1 []int) ([]string, error) {
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)
	query := "SELECT DISTINCT GROUPNAME FROM supplies WHERE GROUPNAME IS NOT NULL" + filterClause + " ORDER BY GROUPNAME"

	rows, err := r.DB.Query(query, filterArgs...)
	if err != nil {
		return nil, fmt.Errorf("error querying groups: %w", err)
	}
	defer rows.Close()

	groups := []string{}
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("error scanning group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (r *SupplyRepository) GetLowStockVisible(threshold int, page, pageSize int, visibleIDX1 []int) ([]Supply, int, error) {
	offset := (page - 1) * pageSize
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	countQuery := `
		SELECT COUNT(*)
		FROM supplies
		WHERE (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) < ?
	` + filterClause
	countArgs := []interface{}{threshold}
	countArgs = append(countArgs, filterArgs...)

	var total int
	if err := r.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("error counting low stock supplies: %w", err)
	}

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) < ?
	` + filterClause + `
		ORDER BY (TONDAUKY + NHAPTRONGKY - XUATTRONGKY) ASC
		LIMIT ? OFFSET ?
	`

	args := []interface{}{threshold}
	args = append(args, filterArgs...)
	args = append(args, pageSize, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying low stock supplies: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning supply: %w", err)
		}
		supplies = append(supplies, s)
	}

	return supplies, total, nil
}

func (r *SupplyRepository) GetForecastCatalogVisible(keyword string, visibleIDX1 []int) ([]Supply, error) {
	searchPattern := "%" + keyword + "%"
	filterClause, filterArgs := buildSupplyVisibilityFilterClause("IDX1", visibleIDX1)

	query := `
		SELECT
			` + supplySelectColumns + `
		FROM supplies
		WHERE (TONDAUKY != 0 OR NHAPTRONGKY != 0 OR XUATTRONGKY != 0 OR TONGNHAP != 0)
	`

	args := make([]interface{}, 0)
	if keyword != "" {
		query += " AND (NAME LIKE ? OR ID LIKE ? OR IDX2 LIKE ? OR MA_HIEU LIKE ?)"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}
	query += filterClause
	query += " ORDER BY IDX1"
	args = append(args, filterArgs...)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying forecast catalog: %w", err)
	}
	defer rows.Close()

	supplies := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning supply: %w", err)
		}
		supplies = append(supplies, s)
	}

	return supplies, nil
}
