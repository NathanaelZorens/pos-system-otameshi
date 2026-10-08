package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func migrate(conn *sql.DB) error {
	data, err := migrationFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := conn.Exec(string(data)); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}
	if err := ensureGuestColumns(conn); err != nil {
		return err
	}
	if err := ensureOrderItemStatus(conn); err != nil {
		return err
	}
	if err := ensureOrdersFulfillmentStatuses(conn); err != nil {
		return err
	}
	if err := ensureGuestClaim(conn); err != nil {
		return err
	}
	if err := ensureCancelReason(conn); err != nil {
		return err
	}
	if err := ensureMenuCategories(conn); err != nil {
		return err
	}
	if err := ensureTakeoutOrders(conn); err != nil {
		return err
	}
	if err := ensureMembers(conn); err != nil {
		return err
	}
	if err := ensureClearedStatus(conn); err != nil {
		return err
	}
	if err := ensurePaidOrderItemsConfirmed(conn); err != nil {
		return err
	}
	if err := ensureServedAt(conn); err != nil {
		return err
	}
	return ensureGuestTokens(conn)
}

// served_at tracks kitchen/service independently of pay (pay-before-served).
func ensureServedAt(conn *sql.DB) error {
	ok, err := hasColumn(conn, "orders", "served_at")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN served_at TEXT`); err != nil {
			return fmt.Errorf("add orders.served_at: %w", err)
		}
	}
	// Status served always means food went out; backfill timestamp.
	if _, err := conn.Exec(`
		UPDATE orders SET served_at = COALESCE(updated_at, created_at)
		WHERE status = 'served' AND served_at IS NULL
	`); err != nil {
		return fmt.Errorf("backfill served_at: %w", err)
	}
	return nil
}

// Line status was added with DEFAULT pending; settled checks should not stay pending.
func ensurePaidOrderItemsConfirmed(conn *sql.DB) error {
	_, err := conn.Exec(`
		UPDATE order_items
		SET status = 'confirmed'
		WHERE status = 'pending'
		  AND order_id IN (
			SELECT id FROM orders WHERE status IN ('paid', 'cleared', 'completed')
		  )
	`)
	if err != nil {
		return fmt.Errorf("confirm settled order lines: %w", err)
	}
	return nil
}

func ensureClearedStatus(conn *sql.DB) error {
	var createSQL sql.NullString
	if err := conn.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='orders'`).Scan(&createSQL); err != nil {
		return err
	}
	if createSQL.Valid && strings.Contains(createSQL.String, "cleared") {
		return nil
	}

	okMember, err := hasColumn(conn, "orders", "member_id")
	if err != nil {
		return err
	}
	okPickup, err := hasColumn(conn, "orders", "pickup_code")
	if err != nil {
		return err
	}

	if _, err := conn.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.Exec(`PRAGMA foreign_keys = ON`)

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE orders_cleared_mig (
			id TEXT PRIMARY KEY,
			dining_table_id TEXT,
			waiter_id TEXT,
			member_id TEXT,
			source TEXT NOT NULL DEFAULT 'staff' CHECK (source IN ('staff', 'guest', 'takeout')),
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'preparing', 'served', 'ready', 'completed', 'paid', 'cleared', 'cancelled')),
			customer_name TEXT,
			guest_claim TEXT,
			cancel_reason TEXT,
			pickup_code TEXT,
			total_cents INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			cancelled_at TEXT
		)
	`); err != nil {
		return err
	}

	cols := `id, dining_table_id, waiter_id, source, status, customer_name, guest_claim, cancel_reason, total_cents, created_at, updated_at, cancelled_at`
	sel := `id, dining_table_id, waiter_id, COALESCE(source, 'staff'), status, customer_name, guest_claim, cancel_reason, total_cents, created_at, updated_at, cancelled_at`
	if okMember && okPickup {
		if _, err := tx.Exec(`
			INSERT INTO orders_cleared_mig (id, dining_table_id, waiter_id, member_id, source, status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at)
			SELECT id, dining_table_id, waiter_id, member_id, COALESCE(source, 'staff'), status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at
			FROM orders
		`); err != nil {
			return err
		}
	} else if okPickup {
		if _, err := tx.Exec(`
			INSERT INTO orders_cleared_mig (id, dining_table_id, waiter_id, member_id, source, status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at)
			SELECT id, dining_table_id, waiter_id, NULL, COALESCE(source, 'staff'), status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at
			FROM orders
		`); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(fmt.Sprintf(`
			INSERT INTO orders_cleared_mig (%s, member_id, pickup_code)
			SELECT %s, NULL, NULL FROM orders
		`, cols+", member_id, pickup_code", sel)); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DROP TABLE orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE orders_cleared_mig RENAME TO orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureMembers(conn *sql.DB) error {
	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS members (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("create members: %w", err)
	}
	ok, err := hasColumn(conn, "orders", "member_id")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN member_id TEXT REFERENCES members(id)`); err != nil {
			return fmt.Errorf("add member_id: %w", err)
		}
	}
	return nil
}

func ensureTakeoutOrders(conn *sql.DB) error {
	var createSQL sql.NullString
	if err := conn.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='orders'`).Scan(&createSQL); err != nil {
		return err
	}
	needsRebuild := !createSQL.Valid ||
		!strings.Contains(createSQL.String, "takeout") ||
		!strings.Contains(createSQL.String, "ready") ||
		!strings.Contains(createSQL.String, "completed") ||
		strings.Contains(createSQL.String, "dining_table_id TEXT NOT NULL")

	okPickup, err := hasColumn(conn, "orders", "pickup_code")
	if err != nil {
		return err
	}

	if !needsRebuild {
		if !okPickup {
			if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN pickup_code TEXT`); err != nil {
				return fmt.Errorf("add pickup_code: %w", err)
			}
		}
		return nil
	}

	if _, err := conn.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.Exec(`PRAGMA foreign_keys = ON`)

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE orders_takeout_mig (
			id TEXT PRIMARY KEY,
			dining_table_id TEXT,
			waiter_id TEXT,
			source TEXT NOT NULL DEFAULT 'staff' CHECK (source IN ('staff', 'guest', 'takeout')),
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'preparing', 'served', 'ready', 'completed', 'paid', 'cleared', 'cancelled')),
			customer_name TEXT,
			guest_claim TEXT,
			cancel_reason TEXT,
			pickup_code TEXT,
			total_cents INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			cancelled_at TEXT
		)
	`); err != nil {
		return err
	}

	hasPickup := okPickup
	if hasPickup {
		if _, err := tx.Exec(`
			INSERT INTO orders_takeout_mig (id, dining_table_id, waiter_id, source, status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at)
			SELECT id, dining_table_id, waiter_id, COALESCE(source, 'staff'), status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at
			FROM orders
		`); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(`
			INSERT INTO orders_takeout_mig (id, dining_table_id, waiter_id, source, status, customer_name, guest_claim, cancel_reason, pickup_code, total_cents, created_at, updated_at, cancelled_at)
			SELECT id, dining_table_id, waiter_id, COALESCE(source, 'staff'), status, customer_name, guest_claim, cancel_reason, NULL, total_cents, created_at, updated_at, cancelled_at
			FROM orders
		`); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DROP TABLE orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE orders_takeout_mig RENAME TO orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_source ON orders(source)`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureMenuCategories(conn *sql.DB) error {
	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS menu_categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("create menu_categories: %w", err)
	}

	ok, err := hasColumn(conn, "menu_items", "category_id")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE menu_items ADD COLUMN category_id TEXT`); err != nil {
			return fmt.Errorf("add category_id: %w", err)
		}
	}

	// Promote free-text category values into menu_categories and link rows.
	rows, err := conn.Query(`
		SELECT DISTINCT TRIM(category) FROM menu_items
		WHERE category IS NOT NULL AND TRIM(category) != ''
		  AND (category_id IS NULL OR category_id = '')
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, name := range names {
		var existingID string
		err := conn.QueryRow(`SELECT id FROM menu_categories WHERE name = ?`, name).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			existingID = uuid.NewString()
			if _, err := conn.Exec(
				`INSERT INTO menu_categories (id, name, sort_order) VALUES (?, ?, ?)`,
				existingID, name, 0,
			); err != nil {
				return fmt.Errorf("insert category %q: %w", name, err)
			}
		} else if err != nil {
			return err
		}
		if _, err := conn.Exec(
			`UPDATE menu_items SET category_id = ? WHERE TRIM(category) = ? AND (category_id IS NULL OR category_id = '')`,
			existingID, name,
		); err != nil {
			return err
		}
	}
	return nil
}

func ensureGuestClaim(conn *sql.DB) error {
	ok, err := hasColumn(conn, "orders", "guest_claim")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN guest_claim TEXT`); err != nil {
			return fmt.Errorf("add guest_claim: %w", err)
		}
	}
	return nil
}

func ensureCancelReason(conn *sql.DB) error {
	ok, err := hasColumn(conn, "orders", "cancel_reason")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN cancel_reason TEXT`); err != nil {
			return fmt.Errorf("add cancel_reason: %w", err)
		}
	}
	return nil
}

func hasColumn(conn *sql.DB, table, column string) (bool, error) {
	rows, err := conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func ensureGuestColumns(conn *sql.DB) error {
	ok, err := hasColumn(conn, "dining_tables", "guest_token")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE dining_tables ADD COLUMN guest_token TEXT`); err != nil {
			return fmt.Errorf("add guest_token: %w", err)
		}
	}
	if _, err := conn.Exec(`CREATE INDEX IF NOT EXISTS idx_dining_tables_guest_token ON dining_tables(guest_token)`); err != nil {
		return fmt.Errorf("index guest_token: %w", err)
	}

	ok, err = hasColumn(conn, "orders", "source")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(`ALTER TABLE orders ADD COLUMN source TEXT NOT NULL DEFAULT 'staff'`); err != nil {
			return fmt.Errorf("add source: %w", err)
		}
	}

	// Old DBs have waiter_id NOT NULL. Rebuild once so guest orders can omit waiter.
	var notNull int
	rows, err := conn.Query(`PRAGMA table_info(orders)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	waiterNotNull := true
	for rows.Next() {
		var cid, nn, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &nn, &dflt, &pk); err != nil {
			return err
		}
		if name == "waiter_id" {
			waiterNotNull = nn == 1
			notNull = nn
		}
	}
	_ = notNull
	if !waiterNotNull {
		return nil
	}

	if _, err := conn.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.Exec(`PRAGMA foreign_keys = ON`)

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE orders_guest_mig (
			id TEXT PRIMARY KEY,
			dining_table_id TEXT NOT NULL,
			waiter_id TEXT,
			source TEXT NOT NULL DEFAULT 'staff',
			status TEXT NOT NULL DEFAULT 'open',
			customer_name TEXT,
			total_cents INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			cancelled_at TEXT
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO orders_guest_mig (id, dining_table_id, waiter_id, source, status, customer_name, total_cents, created_at, updated_at, cancelled_at)
		SELECT id, dining_table_id, waiter_id, COALESCE(source, 'staff'), status, customer_name, total_cents, created_at, updated_at, cancelled_at
		FROM orders
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE orders_guest_mig RENAME TO orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureOrderItemStatus(conn *sql.DB) error {
	ok, err := hasColumn(conn, "order_items", "status")
	if err != nil {
		return err
	}
	if !ok {
		if _, err := conn.Exec(
			`ALTER TABLE order_items ADD COLUMN status TEXT NOT NULL DEFAULT 'pending'`,
		); err != nil {
			return fmt.Errorf("add order_items.status: %w", err)
		}
		// Pre-column lines on settled checks were already sold — mark confirmed.
		if _, err := conn.Exec(`
			UPDATE order_items SET status = 'confirmed'
			WHERE order_id IN (
				SELECT id FROM orders WHERE status IN ('paid', 'cleared', 'completed')
			)
		`); err != nil {
			return fmt.Errorf("backfill order_items.status: %w", err)
		}
	}
	return nil
}

func ensureOrdersFulfillmentStatuses(conn *sql.DB) error {
	var createSQL sql.NullString
	err := conn.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='orders'`).Scan(&createSQL)
	if err != nil {
		return err
	}
	if createSQL.Valid && strings.Contains(createSQL.String, "preparing") {
		return nil
	}
	// Already rebuilt without a narrow CHECK — new statuses are fine.
	if createSQL.Valid && !strings.Contains(createSQL.String, "CHECK (status IN") {
		return nil
	}
	// Old CHECK (open/paid/cancelled only) — rebuild.
	if !(createSQL.Valid && strings.Contains(createSQL.String, "CHECK (status IN")) {
		return nil
	}

	if _, err := conn.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.Exec(`PRAGMA foreign_keys = ON`)

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE orders_fulfill_mig (
			id TEXT PRIMARY KEY,
			dining_table_id TEXT NOT NULL,
			waiter_id TEXT,
			source TEXT NOT NULL DEFAULT 'staff',
			status TEXT NOT NULL DEFAULT 'open',
			customer_name TEXT,
			total_cents INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			cancelled_at TEXT
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO orders_fulfill_mig (id, dining_table_id, waiter_id, source, status, customer_name, total_cents, created_at, updated_at, cancelled_at)
		SELECT id, dining_table_id, waiter_id, COALESCE(source, 'staff'), status, customer_name, total_cents, created_at, updated_at, cancelled_at
		FROM orders
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE orders_fulfill_mig RENAME TO orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureGuestTokens(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT id FROM dining_tables WHERE guest_token IS NULL OR guest_token = ''`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		if _, err := conn.Exec(`UPDATE dining_tables SET guest_token = ? WHERE id = ?`, uuid.NewString(), id); err != nil {
			return err
		}
	}
	return nil
}
