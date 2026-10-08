package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/auth"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
	"github.com/pos-system-otameshi/backend/internal/payment"
	"golang.org/x/crypto/bcrypt"
)

type API struct {
	DB      *sql.DB
	Tokens  *auth.TokenService
	Payment payment.Gateway
}

func isActiveDiningStatus(status string) bool {
	switch status {
	case "open", "preparing", "served":
		return true
	default:
		return false
	}
}

// Table still occupied until Clear table (paid guests may still be seated).
func isTableOccupiedStatus(status string) bool {
	switch status {
	case "open", "preparing", "served", "paid":
		return true
	default:
		return false
	}
}

const sqlTableOccupied = `'open', 'preparing', 'served', 'paid'`

func orderAllowsItemEdit(source, status string) bool {
	if source == "takeout" {
		return status == "open"
	}
	return isActiveDiningStatus(status)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var id, hash, name, role string
	err := a.DB.QueryRow(
		`SELECT id, password_hash, name, role FROM users WHERE email = ?`,
		req.Email,
	).Scan(&id, &hash, &name, &role)
	if err != nil {
		httpjson.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		httpjson.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := a.Tokens.Issue(id, req.Email, name, auth.Role(role))
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "token error")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id": id, "email": req.Email, "name": name, "role": role,
		},
	})
}

func (a *API) Me(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	httpjson.Write(w, http.StatusOK, map[string]any{
		"id": claims.UserID, "email": claims.Email, "name": claims.Name, "role": claims.Role,
	})
}

func (a *API) ListDiningTables(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`SELECT id, label, capacity, is_active, guest_token FROM dining_tables ORDER BY label`)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type tableRow struct {
		ID         string  `json:"id"`
		Label      string  `json:"label"`
		Capacity   int     `json:"capacity"`
		IsActive   bool    `json:"is_active"`
		GuestToken string  `json:"guest_token"`
		GuestPath  string  `json:"guest_path"`
		OpenOrder  *struct {
			ID           string  `json:"id"`
			TotalCents   int     `json:"total_cents"`
			CustomerName *string `json:"customer_name"`
			Source       string  `json:"source"`
			Status       string  `json:"status"`
		} `json:"open_order,omitempty"`
	}
	out := []tableRow{}
	for rows.Next() {
		var t tableRow
		var active int
		var token sql.NullString
		if err := rows.Scan(&t.ID, &t.Label, &t.Capacity, &active, &token); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		t.IsActive = active == 1
		if token.Valid {
			t.GuestToken = token.String
			t.GuestPath = "/t/" + token.String
		}
		var orderID, source, orderStatus string
		var total int
		var cust sql.NullString
		err := a.DB.QueryRow(
			`SELECT id, total_cents, customer_name, COALESCE(source, 'staff'), status
			 FROM orders WHERE dining_table_id = ? AND status IN (`+sqlTableOccupied+`) LIMIT 1`,
			t.ID,
		).Scan(&orderID, &total, &cust, &source, &orderStatus)
		if err == nil {
			o := struct {
				ID           string  `json:"id"`
				TotalCents   int     `json:"total_cents"`
				CustomerName *string `json:"customer_name"`
				Source       string  `json:"source"`
				Status       string  `json:"status"`
			}{ID: orderID, TotalCents: total, Source: source, Status: orderStatus}
			if cust.Valid {
				o.CustomerName = &cust.String
			}
			t.OpenOrder = &o
		}
		out = append(out, t)
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (a *API) ListMenu(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("all") == "1"
	q := `
		SELECT m.id, m.name, m.price_cents, m.category_id, COALESCE(c.name, m.category),
		       m.is_sold_out, m.is_active
		FROM menu_items m
		LEFT JOIN menu_categories c ON c.id = m.category_id
	`
	if !includeInactive {
		q += ` WHERE m.is_active = 1`
	}
	q += ` ORDER BY COALESCE(c.sort_order, 9999), COALESCE(c.name, m.category), m.name`

	rows, err := a.DB.Query(q)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type item struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		PriceCents int     `json:"price_cents"`
		CategoryID *string `json:"category_id"`
		Category   *string `json:"category"`
		IsSoldOut  bool    `json:"is_sold_out"`
		IsActive   bool    `json:"is_active"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var catID, cat sql.NullString
		var sold, active int
		if err := rows.Scan(&it.ID, &it.Name, &it.PriceCents, &catID, &cat, &sold, &active); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		if catID.Valid && catID.String != "" {
			it.CategoryID = &catID.String
		}
		if cat.Valid && cat.String != "" {
			it.Category = &cat.String
		}
		it.IsSoldOut = sold == 1
		it.IsActive = active == 1
		out = append(out, it)
	}
	httpjson.Write(w, http.StatusOK, out)
}

type menuCreateRequest struct {
	Name       string  `json:"name"`
	PriceCents int     `json:"price_cents"`
	CategoryID *string `json:"category_id"`
	Category   *string `json:"category"` // legacy; ignored if category_id set
}

func (a *API) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	var req menuCreateRequest
	if err := httpjson.Decode(r, &req); err != nil || strings.TrimSpace(req.Name) == "" || req.PriceCents < 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	catID, catName, err := a.resolveCategoryID(req.CategoryID)
	if err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if catID == nil && req.Category != nil && strings.TrimSpace(*req.Category) != "" {
		// legacy free-text create: ensure category row exists
		name := strings.TrimSpace(*req.Category)
		var id string
		err := a.DB.QueryRow(`SELECT id FROM menu_categories WHERE name = ? AND is_active = 1`, name).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			id = uuid.NewString()
			if _, err := a.DB.Exec(
				`INSERT INTO menu_categories (id, name, sort_order) VALUES (?, ?, 0)`,
				id, name,
			); err != nil {
				httpjson.Error(w, http.StatusInternalServerError, "db error")
				return
			}
		} else if err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
		catID = &id
		catName = &name
	}

	id := uuid.NewString()
	_, err = a.DB.Exec(
		`INSERT INTO menu_items (id, name, price_cents, category, category_id) VALUES (?, ?, ?, ?, ?)`,
		id, strings.TrimSpace(req.Name), req.PriceCents, catName, catID,
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, http.StatusCreated, map[string]string{"id": id})
}

func (a *API) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req menuCreateRequest
	if err := httpjson.Decode(r, &req); err != nil || strings.TrimSpace(req.Name) == "" || req.PriceCents < 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	catID, catName, err := a.resolveCategoryID(req.CategoryID)
	if err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := a.DB.Exec(
		`UPDATE menu_items SET name = ?, price_cents = ?, category = ?, category_id = ?, updated_at = datetime('now') WHERE id = ? AND is_active = 1`,
		strings.TrimSpace(req.Name), req.PriceCents, catName, catID, id,
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := a.DB.Exec(
		`UPDATE menu_items SET is_active = 0, updated_at = datetime('now') WHERE id = ? AND is_active = 1`,
		id,
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type soldOutRequest struct {
	SoldOut bool `json:"sold_out"`
}

func (a *API) ToggleSoldOut(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req soldOutRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	val := 0
	if req.SoldOut {
		val = 1
	}
	res, err := a.DB.Exec(
		`UPDATE menu_items SET is_sold_out = ?, updated_at = datetime('now') WHERE id = ? AND is_active = 1`,
		val, id,
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type createOrderItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type createOrderRequest struct {
	DiningTableID string                   `json:"dining_table_id"`
	CustomerName  *string                  `json:"customer_name"`
	Items         []createOrderItemRequest `json:"items"`
}

func (a *API) CreateOrder(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	var req createOrderRequest
	if err := httpjson.Decode(r, &req); err != nil || req.DiningTableID == "" || len(req.Items) == 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body: need dining_table_id and at least one item")
		return
	}
	for _, it := range req.Items {
		if it.MenuItemID == "" || it.Quantity < 1 {
			httpjson.Error(w, http.StatusBadRequest, "invalid items")
			return
		}
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var existing string
	err = tx.QueryRow(
		`SELECT id FROM orders WHERE dining_table_id = ? AND status IN (`+sqlTableOccupied+`)`,
		req.DiningTableID,
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
		`INSERT INTO orders (id, dining_table_id, waiter_id, source, customer_name) VALUES (?, ?, ?, 'staff', ?)`,
		orderID, req.DiningTableID, claims.UserID, req.CustomerName,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	// Merge duplicate menu lines from the request
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
	a.writeOrder(w, orderID)
}

func (a *API) GetOrder(w http.ResponseWriter, r *http.Request) {
	a.writeOrder(w, chi.URLParam(r, "id"))
}

func (a *API) writeOrder(w http.ResponseWriter, orderID string) {
	a.writeOrderOnceClaim(w, orderID, "")
}

// writeOrderOnceClaim returns the order JSON; guestClaim is included only when non-empty (create response).
func (a *API) writeOrderOnceClaim(w http.ResponseWriter, orderID, guestClaim string) {
	var o struct {
		ID            string          `json:"id"`
		DiningTableID *string         `json:"dining_table_id"`
		TableLabel    string          `json:"table_label"`
		WaiterID      *string         `json:"waiter_id"`
		WaiterName    string          `json:"waiter_name"`
		Source        string          `json:"source"`
		Status        string          `json:"status"`
		CustomerName  *string         `json:"customer_name"`
		PickupCode    *string         `json:"pickup_code,omitempty"`
		TotalCents    int             `json:"total_cents"`
		CreatedAt     string          `json:"created_at"`
		CancelledAt   *string         `json:"cancelled_at,omitempty"`
		CancelReason  *string         `json:"cancel_reason,omitempty"`
		ServedAt      *string         `json:"served_at,omitempty"`
		Items             []orderItemJSON `json:"items"`
		PendingCount      int             `json:"pending_count"`
		ConfirmedCount    int             `json:"confirmed_count"`
		Payment           *paymentJSON    `json:"payment,omitempty"`
		GuestClaim        string          `json:"guest_claim,omitempty"`
	}
	var cust sql.NullString
	var tableID sql.NullString
	var tableLabel sql.NullString
	var waiterID sql.NullString
	var waiterName sql.NullString
	var cancelledAt sql.NullString
	var cancelReason sql.NullString
	var pickupCode sql.NullString
	var servedAt sql.NullString
	err := a.DB.QueryRow(`
		SELECT o.id, o.dining_table_id, dt.label, o.waiter_id, u.name, COALESCE(o.source, 'staff'), o.status, o.customer_name, o.pickup_code, o.total_cents, o.created_at, o.cancelled_at, o.cancel_reason, o.served_at
		FROM orders o
		LEFT JOIN dining_tables dt ON dt.id = o.dining_table_id
		LEFT JOIN users u ON u.id = o.waiter_id
		WHERE o.id = ?
	`, orderID).Scan(&o.ID, &tableID, &tableLabel, &waiterID, &waiterName, &o.Source, &o.Status, &cust, &pickupCode, &o.TotalCents, &o.CreatedAt, &cancelledAt, &cancelReason, &servedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if tableID.Valid {
		o.DiningTableID = &tableID.String
	}
	if tableLabel.Valid {
		o.TableLabel = tableLabel.String
	} else if o.Source == "takeout" {
		o.TableLabel = "Takeout"
	}
	if waiterID.Valid {
		o.WaiterID = &waiterID.String
	}
	if waiterName.Valid {
		o.WaiterName = waiterName.String
	} else if o.Source == "guest" {
		o.WaiterName = "Guest"
	} else if o.Source == "takeout" {
		o.WaiterName = "Takeout"
	} else {
		o.WaiterName = "—"
	}
	if cust.Valid {
		o.CustomerName = &cust.String
	}
	if pickupCode.Valid && pickupCode.String != "" {
		o.PickupCode = &pickupCode.String
	}
	if cancelledAt.Valid {
		o.CancelledAt = &cancelledAt.String
	}
	if cancelReason.Valid && cancelReason.String != "" {
		o.CancelReason = &cancelReason.String
	}
	if servedAt.Valid && servedAt.String != "" {
		o.ServedAt = &servedAt.String
	}
	o.Items = []orderItemJSON{}
	if guestClaim != "" {
		o.GuestClaim = guestClaim
	}

	rows, err := a.DB.Query(`
		SELECT id, menu_item_id, name_snapshot, unit_price_cents, quantity, COALESCE(status, 'pending')
		FROM order_items WHERE order_id = ? ORDER BY status DESC, name_snapshot
	`, orderID)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var it orderItemJSON
		if err := rows.Scan(&it.ID, &it.MenuItemID, &it.NameSnapshot, &it.UnitPriceCents, &it.Quantity, &it.Status); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		it.LineTotalCents = it.UnitPriceCents * it.Quantity
		o.Items = append(o.Items, it)
		if it.Status == "confirmed" {
			o.ConfirmedCount++
		} else {
			o.PendingCount++
		}
	}

	if o.Status == "paid" || o.Status == "cleared" || o.Source == "takeout" {
		var p paymentJSON
		err := a.DB.QueryRow(`
			SELECT method, amount_cents, provider, provider_ref, paid_at
			FROM payments WHERE order_id = ?
		`, orderID).Scan(&p.Method, &p.AmountCents, &p.Provider, &p.ProviderRef, &p.PaidAt)
		if err == nil {
			o.Payment = &p
		}
	}

	httpjson.Write(w, http.StatusOK, o)
}

type orderItemJSON struct {
	ID             string `json:"id"`
	MenuItemID     string `json:"menu_item_id"`
	NameSnapshot   string `json:"name_snapshot"`
	UnitPriceCents int    `json:"unit_price_cents"`
	Quantity       int    `json:"quantity"`
	LineTotalCents int    `json:"line_total_cents"`
	Status         string `json:"status"` // pending | confirmed
}

type paymentJSON struct {
	Method      string  `json:"method"`
	AmountCents int     `json:"amount_cents"`
	Provider    string  `json:"provider"`
	ProviderRef *string `json:"provider_ref"`
	PaidAt      string  `json:"paid_at"`
}

type addItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

func (a *API) AddOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var req addItemRequest
	if err := httpjson.Decode(r, &req); err != nil || req.MenuItemID == "" || req.Quantity < 1 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status, source string
	if err := tx.QueryRow(`SELECT status, COALESCE(source, 'staff') FROM orders WHERE id = ?`, orderID).Scan(&status, &source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !orderAllowsItemEdit(source, status) {
		httpjson.Error(w, http.StatusConflict, "order is not active")
		return
	}

	var name string
	var price int
	var soldOut, active int
	if err := tx.QueryRow(
		`SELECT name, price_cents, is_sold_out, is_active FROM menu_items WHERE id = ?`,
		req.MenuItemID,
	).Scan(&name, &price, &soldOut, &active); err != nil {
		httpjson.Error(w, http.StatusNotFound, "menu item not found")
		return
	}
	if active != 1 || soldOut == 1 {
		httpjson.Error(w, http.StatusConflict, "item unavailable")
		return
	}

	var existingID string
	var qty int
	// Only merge into an existing pending line — never bump a confirmed line.
	err = tx.QueryRow(
		`SELECT id, quantity FROM order_items WHERE order_id = ? AND menu_item_id = ? AND status = 'pending'`,
		orderID, req.MenuItemID,
	).Scan(&existingID, &qty)
	if err == nil {
		newQty := qty + req.Quantity
		if _, err := tx.Exec(`UPDATE order_items SET quantity = ? WHERE id = ?`, newQty, existingID); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.Exec(
			`INSERT INTO order_items (id, order_id, menu_item_id, name_snapshot, unit_price_cents, quantity, status) VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
			uuid.NewString(), orderID, req.MenuItemID, name, price, req.Quantity,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	} else {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
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

type updateItemRequest struct {
	Quantity int `json:"quantity"`
}

func (a *API) UpdateOrderItem(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	var req updateItemRequest
	if err := httpjson.Decode(r, &req); err != nil || req.Quantity < 0 {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status, source string
	if err := tx.QueryRow(`SELECT status, COALESCE(source, 'staff') FROM orders WHERE id = ?`, orderID).Scan(&status, &source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !orderAllowsItemEdit(source, status) {
		httpjson.Error(w, http.StatusConflict, "order is not active")
		return
	}

	var exists string
	var lineStatus string
	if err := tx.QueryRow(
		`SELECT id, COALESCE(status, 'pending') FROM order_items WHERE id = ? AND order_id = ?`,
		itemID, orderID,
	).Scan(&exists, &lineStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "item not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if lineStatus != "pending" {
		httpjson.Error(w, http.StatusConflict, "confirmed items cannot be changed")
		return
	}

	if req.Quantity == 0 {
		if _, err := tx.Exec(`DELETE FROM order_items WHERE id = ?`, itemID); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	} else {
		if _, err := tx.Exec(`UPDATE order_items SET quantity = ? WHERE id = ?`, req.Quantity, itemID); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := recalcOrderTotal(tx, orderID); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

	// No lines left → cancel so the table becomes Free again (create-on-first-item model).
	var remaining int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM order_items WHERE order_id = ?`, orderID).Scan(&remaining); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if remaining == 0 {
		if _, err := tx.Exec(
			`UPDATE orders SET status = 'cancelled', guest_claim = NULL, cancel_reason = NULL, cancelled_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`,
			orderID,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func recalcOrderTotal(tx *sql.Tx, orderID string) error {
	var total int
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(unit_price_cents * quantity), 0) FROM order_items WHERE order_id = ?
	`, orderID).Scan(&total); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE orders SET total_cents = ?, updated_at = datetime('now') WHERE id = ?`, total, orderID)
	return err
}

func (a *API) ConfirmOrderItems(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status, source string
	if err := tx.QueryRow(`SELECT status, COALESCE(source, 'staff') FROM orders WHERE id = ?`, orderID).Scan(&status, &source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !orderAllowsItemEdit(source, status) {
		httpjson.Error(w, http.StatusConflict, "order is not active")
		return
	}

	res, err := tx.Exec(
		`UPDATE order_items SET status = 'confirmed' WHERE order_id = ? AND status = 'pending'`,
		orderID,
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusConflict, "no pending items to confirm")
		return
	}
	// Dine-in: confirm starts kitchen. Takeout: wait until pay.
	if source != "takeout" {
		if _, err := tx.Exec(
			`UPDATE orders SET status = 'preparing', updated_at = datetime('now') WHERE id = ?`,
			orderID,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) MarkOrderServed(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	tx, err := a.DB.Begin()
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	var status, source string
	var servedAt sql.NullString
	if err := tx.QueryRow(
		`SELECT status, COALESCE(source, 'staff'), served_at FROM orders WHERE id = ?`,
		orderID,
	).Scan(&status, &source, &servedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if source == "takeout" {
		httpjson.Error(w, http.StatusConflict, "use takeout ready for takeout orders")
		return
	}
	if servedAt.Valid && servedAt.String != "" {
		httpjson.Error(w, http.StatusConflict, "already marked served")
		return
	}
	// Pay-before-served: still allow Mark served while paid.
	if status != "preparing" && status != "paid" {
		httpjson.Error(w, http.StatusConflict, "order cannot be marked served in this state")
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
		httpjson.Error(w, http.StatusConflict, "no confirmed items to serve")
		return
	}
	if status == "paid" {
		if _, err := tx.Exec(
			`UPDATE orders SET served_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`,
			orderID,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	} else if _, err := tx.Exec(
		`UPDATE orders SET status = 'served', served_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`,
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

// ClearTable: seats free after pay (guests may have paid while still seated).
func (a *API) ClearTable(w http.ResponseWriter, r *http.Request) {
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
	if source == "takeout" {
		httpjson.Error(w, http.StatusConflict, "clear table is for dine-in only")
		return
	}
	if status != "paid" {
		httpjson.Error(w, http.StatusConflict, "order must be paid before clearing the table")
		return
	}
	if _, err := a.DB.Exec(
		`UPDATE orders SET status = 'cleared', updated_at = datetime('now') WHERE id = ?`,
		orderID,
	); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) markOrderCancelled(orderID, reason string) (bool, error) {
	res, err := a.DB.Exec(
		`UPDATE orders SET status = 'cancelled', guest_claim = NULL, cancel_reason = ?, cancelled_at = datetime('now'), updated_at = datetime('now')
		 WHERE id = ? AND status IN ('open', 'preparing', 'served', 'ready')`,
		reason, orderID,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func decodeCancelReason(r *http.Request) (string, error) {
	var req cancelRequest
	if err := httpjson.Decode(r, &req); err != nil {
		return "", err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return "", errors.New("reason required")
	}
	if len(reason) > 500 {
		return "", errors.New("reason too long")
	}
	return reason, nil
}

// QuitOrder abandons a check before any line is confirmed (guest + staff).
func (a *API) QuitOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	reason, err := decodeCancelReason(r)
	if err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var status string
	if err := a.DB.QueryRow(`SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status); err != nil {
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

	var confirmed int
	if err := a.DB.QueryRow(
		`SELECT COUNT(*) FROM order_items WHERE order_id = ? AND status = 'confirmed'`,
		orderID,
	).Scan(&confirmed); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if confirmed > 0 {
		httpjson.Error(w, http.StatusConflict, "cannot quit after items are confirmed; ask staff to force cancel")
		return
	}

	ok, err := a.markOrderCancelled(orderID, reason)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !ok {
		httpjson.Error(w, http.StatusConflict, "cannot quit")
		return
	}
	a.writeOrder(w, orderID)
}

// ForceCancelOrder is staff-only escape hatch; allowed even with confirmed lines.
func (a *API) ForceCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	reason, err := decodeCancelReason(r)
	if err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ok, err := a.markOrderCancelled(orderID, reason)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if !ok {
		httpjson.Error(w, http.StatusConflict, "cannot force cancel")
		return
	}
	a.writeOrder(w, orderID)
}

type checkoutRequest struct {
	Method string `json:"method"`
}

func (a *API) Checkout(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	var req checkoutRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	method := payment.Method(req.Method)
	switch method {
	case payment.MethodCash, payment.MethodCard, payment.MethodQR, payment.MethodWallet:
	default:
		httpjson.Error(w, http.StatusBadRequest, "invalid payment method")
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
	if err := tx.QueryRow(`SELECT status, COALESCE(source, 'staff'), total_cents FROM orders WHERE id = ?`, orderID).Scan(&status, &source, &total); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if source == "takeout" {
		if status != "open" {
			httpjson.Error(w, http.StatusConflict, "takeout order cannot be paid")
			return
		}
	} else if !isActiveDiningStatus(status) {
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
	if source == "takeout" {
		// Pay-first takeout: kitchen starts after payment.
		if _, err := tx.Exec(
			`UPDATE orders SET status = 'preparing', updated_at = datetime('now') WHERE id = ?`,
			orderID,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "db error")
			return
		}
	} else if _, err := tx.Exec(`UPDATE orders SET status = 'paid', guest_claim = NULL, updated_at = datetime('now') WHERE id = ?`, orderID); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := tx.Commit(); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) ListTransactions(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = from
	}
	toEnd := to + " 23:59:59"

	rows, err := a.DB.Query(`
		SELECT o.id,
			COALESCE(dt.label, CASE WHEN o.source = 'takeout' THEN 'Takeout' ELSE '—' END),
			COALESCE(u.name, CASE
				WHEN o.source = 'guest' THEN 'Guest'
				WHEN o.source = 'takeout' THEN 'Takeout'
				ELSE '—'
			END),
			o.total_cents, p.method, p.paid_at
		FROM orders o
		JOIN payments p ON p.order_id = o.id
		LEFT JOIN dining_tables dt ON dt.id = o.dining_table_id
		LEFT JOIN users u ON u.id = o.waiter_id
		WHERE p.paid_at >= ? AND p.paid_at <= ?
		ORDER BY p.paid_at DESC
	`, from, toEnd)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type row struct {
		OrderID     string `json:"order_id"`
		TableLabel  string `json:"table_label"`
		WaiterName  string `json:"waiter_name"`
		TotalCents  int    `json:"total_cents"`
		Method      string `json:"method"`
		PaidAt      string `json:"paid_at"`
	}
	var out []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.OrderID, &x.TableLabel, &x.WaiterName, &x.TotalCents, &x.Method, &x.PaidAt); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		out = append(out, x)
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (a *API) ReportSummary(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = from
	}
	toEnd := to + " 23:59:59"

	var total int
	var count int
	err := a.DB.QueryRow(`
		SELECT COALESCE(SUM(p.amount_cents), 0), COUNT(*)
		FROM payments p
		JOIN orders o ON o.id = p.order_id
		WHERE p.paid_at >= ? AND p.paid_at <= ?
	`, from, toEnd).Scan(&total, &count)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{
		"from": from, "to": to, "total_cents": total, "order_count": count,
	})
}

func (a *API) ReportItems(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = from
	}
	toEnd := to + " 23:59:59"

	rows, err := a.DB.Query(`
		SELECT oi.name_snapshot, SUM(oi.quantity) AS qty, SUM(oi.quantity * oi.unit_price_cents) AS revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN payments p ON p.order_id = o.id
		WHERE p.paid_at >= ? AND p.paid_at <= ?
		GROUP BY oi.name_snapshot
		ORDER BY qty DESC
	`, from, toEnd)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type row struct {
		Name        string `json:"name"`
		Quantity    int    `json:"quantity"`
		RevenueCents int   `json:"revenue_cents"`
	}
	var out []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.Name, &x.Quantity, &x.RevenueCents); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		out = append(out, x)
	}
	httpjson.Write(w, http.StatusOK, out)
}

// ListCancelledOrders is admin ops (not sales): quit / force / emptied in a date range.
func (a *API) ListCancelledOrders(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = from
	}
	toEnd := to + " 23:59:59"

	rows, err := a.DB.Query(`
		SELECT o.id,
			COALESCE(dt.label, CASE WHEN o.source = 'takeout' THEN 'Takeout' ELSE '—' END),
			COALESCE(o.source, 'staff'),
			COALESCE(u.name, CASE
				WHEN o.source = 'guest' THEN 'Guest'
				WHEN o.source = 'takeout' THEN 'Takeout'
				ELSE '—'
			END),
			o.total_cents, COALESCE(o.cancel_reason, ''), COALESCE(o.cancelled_at, o.updated_at),
			(SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id AND oi.status = 'confirmed'),
			(SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id AND oi.status = 'pending')
		FROM orders o
		LEFT JOIN dining_tables dt ON dt.id = o.dining_table_id
		LEFT JOIN users u ON u.id = o.waiter_id
		WHERE o.status = 'cancelled'
		  AND COALESCE(o.cancelled_at, o.updated_at) >= ?
		  AND COALESCE(o.cancelled_at, o.updated_at) <= ?
		ORDER BY COALESCE(o.cancelled_at, o.updated_at) DESC
	`, from, toEnd)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type row struct {
		OrderID        string `json:"order_id"`
		TableLabel     string `json:"table_label"`
		Source         string `json:"source"`
		WaiterName     string `json:"waiter_name"`
		TotalCents     int    `json:"total_cents"`
		CancelReason   string `json:"cancel_reason"`
		CancelledAt    string `json:"cancelled_at"`
		ConfirmedCount int    `json:"confirmed_count"`
		PendingCount   int    `json:"pending_count"`
	}
	var out []row
	for rows.Next() {
		var x row
		if err := rows.Scan(
			&x.OrderID, &x.TableLabel, &x.Source, &x.WaiterName, &x.TotalCents,
			&x.CancelReason, &x.CancelledAt, &x.ConfirmedCount, &x.PendingCount,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		out = append(out, x)
	}
	if out == nil {
		out = []row{}
	}
	httpjson.Write(w, http.StatusOK, map[string]any{
		"from": from, "to": to,
		"total_count": len(out),
		"orders":      out,
	})
}
