package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/discount"
)

var errMenuItemNotFound = errors.New("menu item not found")

type itemUnavailableError struct {
	Name string
}

func (e *itemUnavailableError) Error() string {
	return "item unavailable: " + e.Name
}

// insertPendingOrderItem loads the menu row, applies an active discount rule, and inserts a pending line.
func insertPendingOrderItem(tx *sql.Tx, orderID, menuItemID string, quantity int) error {
	var name string
	var price int
	var soldOut, active int
	var categoryID sql.NullString
	err := tx.QueryRow(
		`SELECT name, price_cents, is_sold_out, is_active, category_id FROM menu_items WHERE id = ?`,
		menuItemID,
	).Scan(&name, &price, &soldOut, &active, &categoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errMenuItemNotFound
		}
		return err
	}
	if active != 1 || soldOut == 1 {
		return &itemUnavailableError{Name: name}
	}

	var catPtr *string
	if categoryID.Valid && categoryID.String != "" {
		catPtr = &categoryID.String
	}
	applied, err := discount.ResolveUnitPrice(tx, menuItemID, price, catPtr, time.Now())
	if err != nil {
		return fmt.Errorf("resolve discount: %w", err)
	}

	var ruleID any
	var label any
	if applied.RuleID != nil {
		ruleID = *applied.RuleID
	}
	if applied.Label != nil {
		label = *applied.Label
	}

	_, err = tx.Exec(`
		INSERT INTO order_items (
			id, order_id, menu_item_id, name_snapshot,
			unit_price_cents, list_unit_price_cents, discount_rule_id, discount_label,
			quantity, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')
	`, uuid.NewString(), orderID, menuItemID, name,
		applied.UnitPriceCents, applied.ListUnitPriceCents, ruleID, label,
		quantity)
	return err
}
