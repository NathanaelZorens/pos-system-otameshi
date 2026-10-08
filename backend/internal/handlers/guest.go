package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
	"github.com/pos-system-otameshi/backend/internal/payment"
)

func withChiParams(r *http.Request, pairs ...string) *http.Request {
	rctx := chi.NewRouteContext()
	for i := 0; i+1 < len(pairs); i += 2 {
		rctx.URLParams.Add(pairs[i], pairs[i+1])
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

type guestTable struct {
	ID    string
	Label string
	Token string
}

func (a *API) lookupGuestTable(token string) (*guestTable, error) {
	var t guestTable
	err := a.DB.QueryRow(
		`SELECT id, label, guest_token FROM dining_tables WHERE guest_token = ? AND is_active = 1`,
		token,
	).Scan(&t.ID, &t.Label, &t.Token)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (a *API) openOrderForTable(tableID string) (orderID string, source string, status string, claim sql.NullString, err error) {
	err = a.DB.QueryRow(
		`SELECT id, COALESCE(source, 'staff'), status, guest_claim FROM orders WHERE dining_table_id = ? AND status IN ('open', 'preparing', 'served', 'paid') LIMIT 1`,
		tableID,
	).Scan(&orderID, &source, &status, &claim)
	return
}

func newGuestClaim() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func guestClaimFromRequest(r *http.Request) string {
	return r.Header.Get("X-Guest-Claim")
}

func claimMatches(stored sql.NullString, provided string) bool {
	if !stored.Valid || stored.String == "" || provided == "" {
		return false
	}
	a := []byte(stored.String)
	b := []byte(provided)
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// GuestTableStatus: free → draft; matching claim → resume; else blocked.
func (a *API) GuestTableStatus(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := a.lookupGuestTable(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "table not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	orderID, source, status, claim, err := a.openOrderForTable(t.ID)
	if errors.Is(err, sql.ErrNoRows) {
		httpjson.Write(w, http.StatusOK, map[string]any{
			"table_id":    t.ID,
			"table_label": t.Label,
			"guest_token": t.Token,
			"blocked":     false,
		})
		return
	}
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	if claimMatches(claim, guestClaimFromRequest(r)) {
		httpjson.Write(w, http.StatusOK, map[string]any{
			"table_id":         t.ID,
			"table_label":      t.Label,
			"guest_token":      t.Token,
			"blocked":          false,
			"resume_order_id":  orderID,
			"open_order_source": source,
		})
		return
	}

	blockedReason := "This table already has an open order. Please ask staff for help."
	if status == "paid" {
		blockedReason = "This table is paid but still occupied. Ask staff to clear the table."
	}

	httpjson.Write(w, http.StatusOK, map[string]any{
		"table_id":          t.ID,
		"table_label":       t.Label,
		"guest_token":       t.Token,
		"blocked":           true,
		"blocked_reason":    blockedReason,
		"open_order_id":     orderID,
		"open_order_source": source,
	})
}

func (a *API) GuestMenu(w http.ResponseWriter, r *http.Request) {
	if _, err := a.lookupGuestTable(chi.URLParam(r, "token")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "table not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	// Reuse staff menu listing shape (active only).
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Del("all")
	r2.URL.RawQuery = q.Encode()
	a.ListMenu(w, r2)
}

func (a *API) GuestCreateOrder(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := a.lookupGuestTable(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "table not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	if _, _, _, _, err := a.openOrderForTable(t.ID); err == nil {
		httpjson.Error(w, http.StatusConflict, "table already has an open order")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	var req createOrderRequest
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

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	// Re-check inside transaction
	var existing string
	err = tx.QueryRow(
		`SELECT id FROM orders WHERE dining_table_id = ? AND status IN ('open', 'preparing', 'served', 'paid')`,
		t.ID,
	).Scan(&existing)
	if err == nil {
		httpjson.Error(w, http.StatusConflict, "table already has an open order")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	orderID := uuid.NewString()
	if _, err := tx.Exec(
		`INSERT INTO orders (id, dining_table_id, waiter_id, source, customer_name, guest_claim) VALUES (?, ?, NULL, 'guest', ?, ?)`,
		orderID, t.ID, req.CustomerName, claim,
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
		var name string
		var price int
		var soldOut, active int
		if err := tx.QueryRow(
			`SELECT name, price_cents, is_sold_out, is_active FROM menu_items WHERE id = ?`,
			menuItemID,
		).Scan(&name, &price, &soldOut, &active); err != nil {
			httpjson.Error(w, http.StatusNotFound, "menu item not found")
			return
		}
		if active != 1 || soldOut == 1 {
			httpjson.Error(w, http.StatusConflict, "item unavailable: "+name)
			return
		}
		if _, err := tx.Exec(
			`INSERT INTO order_items (id, order_id, menu_item_id, name_snapshot, unit_price_cents, quantity, status) VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
			uuid.NewString(), orderID, menuItemID, name, price, qty,
		); err != nil {
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

// guestOrderAccess: table token + guest source. Active orders need matching claim;
// paid/cancelled allow read without claim (receipt).
func (a *API) guestOrderAccess(orderID, token, providedClaim string, mutate bool) (ok bool, err error) {
	var source string
	var status string
	var claim sql.NullString
	err = a.DB.QueryRow(`
		SELECT COALESCE(o.source, 'staff'), o.status, o.guest_claim
		FROM orders o
		JOIN dining_tables dt ON dt.id = o.dining_table_id
		WHERE o.id = ? AND dt.guest_token = ?
	`, orderID, token).Scan(&source, &status, &claim)
	if err != nil {
		return false, err
	}
	if source != "guest" {
		return false, nil
	}
	active := isActiveDiningStatus(status)
	if active || mutate {
		if !claimMatches(claim, providedClaim) {
			return false, nil
		}
	}
	return true, nil
}

func (a *API) requireGuestOrder(w http.ResponseWriter, r *http.Request, mutate bool) (orderID string, ok bool) {
	token := chi.URLParam(r, "token")
	orderID = chi.URLParam(r, "orderId")
	allowed, err := a.guestOrderAccess(orderID, token, guestClaimFromRequest(r), mutate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return "", false
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return "", false
	}
	if !allowed {
		// Wrong/missing claim on an existing guest order → forbidden (not "not found")
		httpjson.Error(w, http.StatusForbidden, "guest claim required")
		return "", false
	}
	return orderID, true
}

func (a *API) GuestGetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, false)
	if !ok {
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) GuestAddOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.AddOrderItem(w, r)
}

func (a *API) GuestUpdateOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, true)
	if !ok {
		return
	}
	itemID := chi.URLParam(r, "itemId")
	r = withChiParams(r, "id", orderID, "itemId", itemID)
	a.UpdateOrderItem(w, r)
}

func (a *API) GuestConfirmOrderItems(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.ConfirmOrderItems(w, r)
}

func (a *API) GuestQuitOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.QuitOrder(w, r)
}

func (a *API) GuestCheckout(w http.ResponseWriter, r *http.Request) {
	orderID, ok := a.requireGuestOrder(w, r, true)
	if !ok {
		return
	}

	var req checkoutRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	method := payment.Method(req.Method)
	switch method {
	case payment.MethodCard, payment.MethodQR, payment.MethodWallet:
	default:
		httpjson.Error(w, http.StatusBadRequest, "guest checkout supports card, qr, or wallet only")
		return
	}
	a.guestCheckout(w, r, orderID, method)
}

func (a *API) guestCheckout(w http.ResponseWriter, r *http.Request, orderID string, method payment.Method) {
	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status string
	var total int
	if err := tx.QueryRow(`SELECT status, total_cents FROM orders WHERE id = ?`, orderID).Scan(&status, &total); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !isActiveDiningStatus(status) {
		httpjson.Error(w, http.StatusConflict, "order is not active")
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
		`UPDATE orders SET status = 'paid', guest_claim = NULL, updated_at = datetime('now') WHERE id = ?`,
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
