package models

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type OrderUnreadSnapshot struct {
	HasSupplierRedDot bool     `json:"hasSupplierRedDot"`
	UnreadGroupKeys   []string `json:"unreadGroupKeys"`
}

type OrderUnreadRepository struct {
	DB *sql.DB
}

func NewOrderUnreadRepository(db *sql.DB) *OrderUnreadRepository {
	return &OrderUnreadRepository{DB: db}
}

func (r *OrderUnreadRepository) EnsureSchema() error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS order_group_reads (
			user_id BIGINT NOT NULL,
			group_key VARCHAR(255) NOT NULL,
			seen_at DATETIME NOT NULL,
			PRIMARY KEY (user_id, group_key),
			KEY idx_group_reads_user_seen (user_id, seen_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
		`
		CREATE TABLE IF NOT EXISTS supplier_alert_reads (
			user_id BIGINT NOT NULL PRIMARY KEY,
			last_seen_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
	}

	for _, statement := range statements {
		if _, err := r.DB.Exec(statement); err != nil {
			return fmt.Errorf("error ensuring unread schema: %w", err)
		}
	}

	return nil
}

func (r *OrderUnreadRepository) GetUnreadSnapshot(userID int64) (*OrderUnreadSnapshot, error) {
	lastSeenAt := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = r.DB.QueryRow(`SELECT last_seen_at FROM supplier_alert_reads WHERE user_id = ?`, userID).Scan(&lastSeenAt)

	var hasRedDot int
	if err := r.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM pending_orders
			WHERE source = ?
			  AND created_at_ts >= DATE_SUB(?, INTERVAL 1 SECOND)
			LIMIT 1
		)
	`, OrderSourceForecast, lastSeenAt).Scan(&hasRedDot); err != nil {
		return nil, fmt.Errorf("error checking supplier red dot: %w", err)
	}

	rows, err := r.DB.Query(`
		SELECT DISTINCT p.group_key
		FROM pending_orders p
		LEFT JOIN order_group_reads r
			ON r.user_id = ? AND r.group_key = p.group_key
		WHERE p.source = ?
		  AND p.group_key <> ''
		  AND r.group_key IS NULL
	`, userID, OrderSourceForecast)
	if err != nil {
		return nil, fmt.Errorf("error loading unread group keys: %w", err)
	}
	defer rows.Close()

	groupKeys := make([]string, 0)
	for rows.Next() {
		var groupKey string
		if err := rows.Scan(&groupKey); err != nil {
			return nil, fmt.Errorf("error scanning unread group key: %w", err)
		}
		groupKeys = append(groupKeys, groupKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unread group keys: %w", err)
	}

	sort.Strings(groupKeys)

	return &OrderUnreadSnapshot{
		HasSupplierRedDot: hasRedDot == 1,
		UnreadGroupKeys:   groupKeys,
	}, nil
}

func (r *OrderUnreadRepository) MarkSupplierAlertSeen(userID int64, _ time.Time) error {
	if _, err := r.DB.Exec(`
		INSERT INTO supplier_alert_reads (user_id, last_seen_at, updated_at)
		VALUES (?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE
			last_seen_at = VALUES(last_seen_at),
			updated_at = VALUES(updated_at)
	`, userID); err != nil {
		return fmt.Errorf("error marking supplier alert seen: %w", err)
	}
	return nil
}

func (r *OrderUnreadRepository) MarkGroupsSeen(userID int64, groupKeys []string, _ time.Time) error {
	if len(groupKeys) == 0 {
		return nil
	}

	tx, err := r.DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting group seen transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO order_group_reads (user_id, group_key, seen_at)
		VALUES (?, ?, NOW())
		ON DUPLICATE KEY UPDATE seen_at = VALUES(seen_at)
	`)
	if err != nil {
		return fmt.Errorf("error preparing mark group seen statement: %w", err)
	}
	defer stmt.Close()

	for _, groupKey := range groupKeys {
		if groupKey == "" {
			continue
		}
		if _, err := stmt.Exec(userID, groupKey); err != nil {
			return fmt.Errorf("error marking group seen: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing mark group seen: %w", err)
	}

	return nil
}
