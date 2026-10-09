package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/auth"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
	"golang.org/x/crypto/bcrypt"
)

type memberAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (a *API) MemberRegister(w http.ResponseWriter, r *http.Request) {
	var req memberAuthRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	name := strings.TrimSpace(req.Name)
	if email == "" || req.Password == "" || name == "" {
		httpjson.Error(w, http.StatusBadRequest, "email, password, and name required")
		return
	}
	if len(req.Password) < 6 {
		httpjson.Error(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "hash error")
		return
	}
	id := uuid.NewString()
	if _, err := a.DB.Exec(
		`INSERT INTO members (id, email, password_hash, name) VALUES (?, ?, ?, ?)`,
		id, email, string(hash), name,
	); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			httpjson.Error(w, http.StatusConflict, "email already registered")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeMemberToken(w, id, email, name)
}

func (a *API) MemberLogin(w http.ResponseWriter, r *http.Request) {
	var req memberAuthRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))

	var id, hash, name string
	err := a.DB.QueryRow(
		`SELECT id, password_hash, name FROM members WHERE email = ?`,
		email,
	).Scan(&id, &hash, &name)
	if err != nil {
		httpjson.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		httpjson.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	a.writeMemberToken(w, id, email, name)
}

func (a *API) writeMemberToken(w http.ResponseWriter, id, email, name string) {
	token, err := a.Tokens.Issue(id, email, name, auth.RoleMember)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "token error")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{
		"token": token,
		"member": map[string]any{
			"id": id, "email": email, "name": name, "role": string(auth.RoleMember),
		},
	})
}

func (a *API) MemberMe(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	httpjson.Write(w, http.StatusOK, map[string]any{
		"id": claims.UserID, "email": claims.Email, "name": claims.Name, "role": claims.Role,
	})
}

func (a *API) memberLiveTakeoutID(memberID string) (orderID string, err error) {
	err = a.DB.QueryRow(`
		SELECT id FROM orders
		WHERE member_id = ? AND source = 'takeout' AND status IN ('open', 'preparing', 'ready')
		ORDER BY created_at DESC LIMIT 1
	`, memberID).Scan(&orderID)
	return
}

func (a *API) requireMemberTakeout(w http.ResponseWriter, r *http.Request, mutate bool) (orderID string, memberID string, ok bool) {
	claims, authed := auth.ClaimsFromRequest(r)
	if !authed || claims.Role != auth.RoleMember {
		httpjson.Error(w, http.StatusUnauthorized, "unauthorized")
		return "", "", false
	}
	memberID = claims.UserID
	orderID = chi.URLParam(r, "orderId")

	var source, status string
	var owner sql.NullString
	err := a.DB.QueryRow(
		`SELECT COALESCE(source, 'staff'), status, member_id FROM orders WHERE id = ?`,
		orderID,
	).Scan(&source, &status, &owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return "", "", false
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return "", "", false
	}
	if source != "takeout" || !owner.Valid || owner.String != memberID {
		httpjson.Error(w, http.StatusForbidden, "not your order")
		return "", "", false
	}
	live := status == "open" || status == "preparing" || status == "ready"
	if mutate {
		if status != "open" {
			httpjson.Error(w, http.StatusConflict, "order cannot be edited")
			return "", "", false
		}
		return orderID, memberID, true
	}
	if !live && status != "completed" && status != "cancelled" {
		httpjson.Error(w, http.StatusConflict, "order not available")
		return "", "", false
	}
	return orderID, memberID, true
}

// MemberTakeoutActive returns the member's live takeout order, if any.
func (a *API) MemberTakeoutActive(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	orderID, err := a.memberLiveTakeoutID(claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		httpjson.Write(w, http.StatusOK, map[string]any{"order_id": nil})
		return
	}
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"order_id": orderID})
}

// MemberTakeoutHistory: completed / cancelled takeout for this member (newest first).
func (a *API) MemberTakeoutHistory(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	rows, err := a.DB.Query(`
		SELECT o.id, o.status, o.pickup_code, o.customer_name, o.total_cents, o.created_at, o.updated_at,
			(SELECT p.method FROM payments p WHERE p.order_id = o.id LIMIT 1),
			(SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.id)
		FROM orders o
		WHERE o.member_id = ? AND o.source = 'takeout'
		  AND o.status IN ('completed', 'cancelled')
		ORDER BY o.created_at DESC
		LIMIT 40
	`, claims.UserID)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type row struct {
		ID           string  `json:"id"`
		Status       string  `json:"status"`
		PickupCode   *string `json:"pickup_code"`
		CustomerName *string `json:"customer_name"`
		TotalCents   int     `json:"total_cents"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
		PayMethod    *string `json:"pay_method,omitempty"`
		ItemCount    int     `json:"item_count"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		var code, cust, method sql.NullString
		if err := rows.Scan(
			&x.ID, &x.Status, &code, &cust, &x.TotalCents, &x.CreatedAt, &x.UpdatedAt, &method, &x.ItemCount,
		); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		if code.Valid && code.String != "" {
			x.PickupCode = &code.String
		}
		if cust.Valid {
			x.CustomerName = &cust.String
		}
		if method.Valid {
			x.PayMethod = &method.String
		}
		out = append(out, x)
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (a *API) MemberTakeoutCreate(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromRequest(r)
	memberID := claims.UserID

	if existing, err := a.memberLiveTakeoutID(memberID); err == nil {
		httpjson.Write(w, http.StatusConflict, map[string]any{
			"error":           "you already have an active takeout order",
			"resume_order_id": existing,
		})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}

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

	cust := req.CustomerName
	if cust == nil || strings.TrimSpace(*cust) == "" {
		n := claims.Name
		cust = &n
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
		`INSERT INTO orders (id, dining_table_id, waiter_id, member_id, source, customer_name, guest_claim, pickup_code)
		 VALUES (?, NULL, NULL, ?, 'takeout', ?, NULL, ?)`,
		orderID, memberID, cust, code,
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

func (a *API) MemberTakeoutGet(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, false)
	if !ok {
		return
	}
	a.writeOrder(w, orderID)
}

func (a *API) MemberTakeoutAddItem(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.AddOrderItem(w, r)
}

func (a *API) MemberTakeoutUpdateItem(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, true)
	if !ok {
		return
	}
	itemID := chi.URLParam(r, "itemId")
	r = withChiParams(r, "id", orderID, "itemId", itemID)
	a.UpdateOrderItem(w, r)
}

func (a *API) MemberTakeoutConfirm(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.ConfirmOrderItems(w, r)
}

func (a *API) MemberTakeoutQuit(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, true)
	if !ok {
		return
	}
	r = withChiParams(r, "id", orderID)
	a.QuitOrder(w, r)
}

func (a *API) MemberTakeoutCheckout(w http.ResponseWriter, r *http.Request) {
	orderID, _, ok := a.requireMemberTakeout(w, r, true)
	if !ok {
		return
	}
	a.takeoutCashlessCheckout(w, r, orderID)
}
