package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/auth"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
	"github.com/pos-system-otameshi/backend/internal/payment"
)

func newPickupCode() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)[:4]), nil
}

func (a *API) takeoutOrderAccess(orderID, providedClaim string, mutate bool) (ok bool, status string, err error) {
	var source string
	var claim sql.NullString
	err = a.DB.QueryRow(
		`SELECT COALESCE(source, 'staff'), status, guest_claim FROM orders WHERE id = ?`,
		orderID,
	).Scan(&source, &status, &claim)
	if err != nil {
		return false, "", err
	}
	if source != "takeout" {
		return false, status, nil
	}
	live := status == "open" || status == "preparing" || status == "ready"
	if mutate {
		if !live || !claimMatches(claim, providedClaim) {
			return false, status, nil
		}
		return true, status, nil
	}
	// Claim required while the order is live; terminal states are readable by id alone.
	if live && !claimMatches(claim, providedClaim) {
		return false, status, nil
	}
	return true, status, nil
}

func (a *API) requireTakeoutGuest(w http.ResponseWriter, r *http.Request, mutate bool) (orderID string, ok bool) {
	orderID = chi.URLParam(r, "orderId")
	allowed, _, err := a.takeoutOrderAccess(orderID, guestClaimFromRequest(r), mutate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return "", false
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return "", false
	}
	if !allowed {
		httpjson.Error(w, http.StatusForbidden, "guest claim required")
		return "", false
	}
	return orderID, true
}

func (a *API) TakeoutMenu(w http.ResponseWriter, r *http.Request) {
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Del("all")
	r2.URL.RawQuery = q.Encode()
	a.ListMenu(w, r2)
}

type takeoutCreateRequest struct {
	CustomerName *string                  `json:"customer_name"`
	Items        []createOrderItemRequest `json:"items"`
}

func (a *API) TakeoutCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req takeoutCreateRequest
	if err := httpjson.Decode(r, &req); err != nil || len(req.Items) == 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body: need at least one item")
		return
	}
	for _, it := range req.Items {
		if it.MenuItemID == "" || it.Quantity < 1 {
			httpjson.Error(w, http.StatusBadRequest, "invalid items")
			return
		}
	}

	claim, err := newGuestClaim()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "claim error")
		return
	}
	code, err := newPickupCode()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "pickup code error")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	orderID := uuid.NewString()
	if _, err := tx.Exec(
		`INSERT INTO orders (id, dining_table_id, waiter_id, source, customer_name, guest_claim, pickup_code)
		 VALUES (?, NULL, NULL, 'takeout', ?, ?, ?)`,
		orderID, req.CustomerName, claim, code,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	qtyByItem := map[string]int{}
	orderKeys := []string{}
	for _, it := range req.Items {
		if _, ok := qtyByItem[it.MenuItemID]; !ok {
			orderKeys = append(orderKeys, it.MenuItemID)
		}
		qtyByItem[it.MenuItemID] += it.Quantity
	}
	for _, menuItemID := range orderKeys {
		qty := qtyByItem[menuItemID]
		if err := insertPendingOrderItem(tx, orderID, menuItemID, qty); err != nil {
			var unavail *itemUnavailableError
			if errors.As(err, &unavail) {
				httpjson.Error(w, http.StatusConflict, unavail.Error())
				return
			}
			if errors.Is(err, errMenuItemNotFound) {
				httpjson.Error(w, http.StatusNotFound, "menu item not found")
				return
			}
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	}
	if err := recalcOrderTotal(tx, orderID); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrderOnceClaim(w, orderID, claim)
}

// StaffCreateTakeoutOrder: counter-order takeout (waiter/admin). No guest claim; any pay method at checkout.
func (a *API) StaffCreateTakeoutOrder(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	var req takeoutCreateRequest
	if err := httpjson.Decode(r, &req); err != nil || len(req.Items) == 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body: need at least one item")
		return
	}
	for _, it := range req.Items {
		if it.MenuItemID == "" || it.Quantity < 1 {
			httpjson.Error(w, http.StatusBadRequest, "invalid items")
			return
		}
	}

	code, err := newPickupCode()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "pickup code error")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	orderID := uuid.NewString()
	if _, err := tx.Exec(
		`INSERT INTO orders (id, dining_table_id, waiter_id, source, customer_name, guest_claim, pickup_code)
		 VALUES (?, NULL, ?, 'takeout', ?, NULL, ?)`,
		orderID, claims.UserID, req.CustomerName, code,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	qtyByItem := map[string]int{}
	orderKeys := []string{}
	for _, it := range req.Items {
		if _, ok := qtyByItem[it.MenuItemID]; !ok {
			orderKeys = append(orderKeys, it.MenuItemID)
		}
		qtyByItem[it.MenuItemID] += it.Quantity
	}
	for _, menuItemID := range orderKeys {
		qty := qtyByItem[menuItemID]
		if err := insertPendingOrderItem(tx, orderID, menuItemID, qty); err != nil {
			var unavail *itemUnavailableError
			if errors.As(err, &unavail) {
				httpjson.Error(w, http.StatusConflict, unavail.Error())
				return
			}
			if errors.Is(err, errMenuItemNotFound) {
				httpjson.Error(w, http.StatusNotFound, "menu item not found")
				return
			}
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	}
	if err := recalcOrderTotal(tx, orderID); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) TakeoutGetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, false)
	if !ok {
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) TakeoutAddOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.AddOrderItem(w, r)
}

func (a *API) TakeoutUpdateOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, true)
	if !ok {
		return
	}
	itemID := chi.URLParam(r, "itemId")
	r = withChiParams(r, "id", orderID, "itemId", itemID)
	a.UpdateOrderItem(w, r)
}

func (a *API) TakeoutConfirmOrderItems(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.ConfirmOrderItems(w, r)
}

func (a *API) TakeoutQuitOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.QuitOrder(w, r)
}

func (a *API) TakeoutCheckout(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireTakeoutGuest(w, r, true)
	if !ok {
		return
	}
	a.takeoutCashlessCheckout(w, r, orderID)
}

// takeoutCashlessCheckout: pay-first takeout (card/qr/wallet) → preparing.
func (a *API) takeoutCashlessCheckout(w http.ResponseWriter, r *http.Request, orderID string) {
	var req checkoutRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	method := payment.Method(req.Method)
	switch method {
	case payment.MethodCard, payment.MethodQR, payment.MethodWallet:
	default:
		httpjson.Error(w, http.StatusBadRequest, "takeout checkout supports card, qr, or wallet only")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status, source string
	var total int
	if err := tx.QueryRow(
		`SELECT status, COALESCE(source, 'staff'), total_cents FROM orders WHERE id = ?`,
		orderID,
	).Scan(&status, &source, &total); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if source != "takeout" || status != "open" {
		httpjson.Error(w, http.StatusConflict, "order cannot be paid")
		return
	}
	if total <= 0 {
		httpjson.Error(w, http.StatusConflict, "order is empty")
		return
	}

	var pending int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM order_items WHERE order_id = ? AND status = 'pending'`,
		orderID,
	).Scan(&pending); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if pending > 0 {
		httpjson.Error(w, http.StatusConflict, "confirm pending items before paying")
		return
	}
	var confirmed int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM order_items WHERE order_id = ? AND status = 'confirmed'`,
		orderID,
	).Scan(&confirmed); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if confirmed == 0 {
		httpjson.Error(w, http.StatusConflict, "no confirmed items to pay for")
		return
	}

	result, err := a.Payment.Charge(r.Context(), payment.ChargeRequest{
		OrderID: orderID, AmountCents: total, Method: method,
	})
	if err != nil {
		httpjson.Error(w, http.StatusPaymentRequired, err.Error())
		return
	}

	paymentID := uuid.NewString()
	var ref *string
	if result.ProviderRef != "" {
		ref = &result.ProviderRef
	}
	if _, err := tx.Exec(`
		INSERT INTO payments (id, order_id, method, amount_cents, status, provider, provider_ref)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		paymentID, orderID, string(method), total, result.Status, result.Provider, ref,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if _, err := tx.Exec(
		`UPDATE orders SET status = 'preparing', updated_at = datetime('now') WHERE id = ?`,
		orderID,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

// ListTakeoutQueue: staff view of preparing + ready takeout orders.
func (a *API) ListTakeoutQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT o.id, o.status, o.customer_name, o.pickup_code, o.total_cents, o.created_at, o.updated_at,
			(SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id)
		FROM orders o
		WHERE o.source = 'takeout' AND o.status IN ('preparing', 'ready')
		ORDER BY
			CASE o.status WHEN 'ready' THEN 0 WHEN 'preparing' THEN 1 ELSE 2 END,
			o.updated_at
	`)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type row struct {
		ID           string  `json:"id"`
		Status       string  `json:"status"`
		CustomerName *string `json:"customer_name"`
		PickupCode   *string `json:"pickup_code"`
		TotalCents   int     `json:"total_cents"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
		ItemCount    int     `json:"item_count"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		var cust, code sql.NullString
		if err := rows.Scan(&x.ID, &x.Status, &cust, &code, &x.TotalCents, &x.CreatedAt, &x.UpdatedAt, &x.ItemCount); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		if cust.Valid {
			x.CustomerName = &cust.String
		}
		if code.Valid {
			x.PickupCode = &code.String
		}
		out = append(out, x)
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (a *API) MarkTakeoutReady(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var source, status string
	if err := a.DB.QueryRow(
		`SELECT COALESCE(source, 'staff'), status FROM orders WHERE id = ?`,
		orderID,
	).Scan(&source, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if source != "takeout" {
		httpjson.Error(w, http.StatusConflict, "not a takeout order")
		return
	}
	if status != "preparing" {
		httpjson.Error(w, http.StatusConflict, "order must be preparing to mark ready")
		return
	}
	if _, err := a.DB.Exec(
		`UPDATE orders SET status = 'ready', updated_at = datetime('now') WHERE id = ?`,
		orderID,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) MarkTakeoutCompleted(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var source, status string
	if err := a.DB.QueryRow(
		`SELECT COALESCE(source, 'staff'), status FROM orders WHERE id = ?`,
		orderID,
	).Scan(&source, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if source != "takeout" {
		httpjson.Error(w, http.StatusConflict, "not a takeout order")
		return
	}
	if status != "ready" {
		httpjson.Error(w, http.StatusConflict, "order must be ready to complete")
		return
	}
	if _, err := a.DB.Exec(
		`UPDATE orders SET status = 'completed', guest_claim = NULL, updated_at = datetime('now') WHERE id = ?`,
		orderID,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}
