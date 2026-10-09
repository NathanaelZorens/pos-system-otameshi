package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/discount"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
)

type discountRuleJSON struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	TargetType      string  `json:"target_type"`
	TargetID        string  `json:"target_id"`
	TargetName      string  `json:"target_name,omitempty"`
	DiscountType    string  `json:"discount_type"`
	Amount          int     `json:"amount"`
	StartTime       *string `json:"start_time"`
	EndTime         *string `json:"end_time"`
	Weekdays        *string `json:"weekdays"`
	StartsOn        *string `json:"starts_on"`
	EndsOn          *string `json:"ends_on"`
	IsActive        bool    `json:"is_active"`
	IsFeaturedPrice bool    `json:"is_featured_price"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type discountRuleRequest struct {
	Name            string  `json:"name"`
	TargetType      string  `json:"target_type"`
	TargetID        string  `json:"target_id"`
	DiscountType    string  `json:"discount_type"`
	Amount          int     `json:"amount"`
	StartTime       *string `json:"start_time"`
	EndTime         *string `json:"end_time"`
	Weekdays        *string `json:"weekdays"`
	StartsOn        *string `json:"starts_on"`
	EndsOn          *string `json:"ends_on"`
	IsActive        *bool   `json:"is_active"`
	IsFeaturedPrice *bool   `json:"is_featured_price"`
}

func (a *API) ListDiscountRules(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT r.id, r.name, r.target_type, r.target_id, r.discount_type, r.amount,
		       r.start_time, r.end_time, r.weekdays, r.starts_on, r.ends_on, r.is_active,
		       COALESCE(r.is_featured_price, 0),
		       r.created_at, r.updated_at,
		       CASE
		         WHEN r.target_type = 'item' THEN (SELECT name FROM menu_items WHERE id = r.target_id)
		         WHEN r.target_type = 'category' THEN (SELECT name FROM menu_categories WHERE id = r.target_id)
		         ELSE NULL
		       END
		FROM discount_rules r
		ORDER BY r.updated_at DESC, r.name
	`)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	out := []discountRuleJSON{}
	for rows.Next() {
		rule, err := scanDiscountRule(rows)
		if err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		out = append(out, rule)
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (a *API) CreateDiscountRule(w http.ResponseWriter, r *http.Request) {
	var req discountRuleRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateDiscountRule(req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.assertDiscountTarget(req.TargetType, req.TargetID); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	id := uuid.NewString()
	active := 1
	if req.IsActive != nil && !*req.IsActive {
		active = 0
	}
	featured := featuredInt(req)
	_, err := a.DB.Exec(`
		INSERT INTO discount_rules (
			id, name, target_type, target_id, discount_type, amount,
			start_time, end_time, weekdays, starts_on, ends_on, is_active, is_featured_price
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, strings.TrimSpace(req.Name), req.TargetType, req.TargetID, req.DiscountType, req.Amount,
		nullStr(req.StartTime), nullStr(req.EndTime), nullStr(req.Weekdays),
		nullStr(req.StartsOn), nullStr(req.EndsOn), active, featured)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	a.writeDiscountRule(w, id, http.StatusCreated)
}

func (a *API) UpdateDiscountRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req discountRuleRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := validateDiscountRule(req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.assertDiscountTarget(req.TargetType, req.TargetID); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	active := 1
	if req.IsActive != nil && !*req.IsActive {
		active = 0
	}
	featured := featuredInt(req)
	res, err := a.DB.Exec(`
		UPDATE discount_rules SET
			name = ?, target_type = ?, target_id = ?, discount_type = ?, amount = ?,
			start_time = ?, end_time = ?, weekdays = ?, starts_on = ?, ends_on = ?,
			is_active = ?, is_featured_price = ?, updated_at = datetime('now')
		WHERE id = ?
	`, strings.TrimSpace(req.Name), req.TargetType, req.TargetID, req.DiscountType, req.Amount,
		nullStr(req.StartTime), nullStr(req.EndTime), nullStr(req.Weekdays),
		nullStr(req.StartsOn), nullStr(req.EndsOn), active, featured, id)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	a.writeDiscountRule(w, id, http.StatusOK)
}

func (a *API) DeleteDiscountRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := a.DB.Exec(`DELETE FROM discount_rules WHERE id = ?`, id)
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

type featuredPreviewRequest struct {
	ExcludeRuleID string `json:"exclude_rule_id"`
	TargetType    string `json:"target_type"`
	TargetID      string `json:"target_id"`
	DiscountType  string `json:"discount_type"`
	Amount        int    `json:"amount"`
}

func (a *API) PreviewFeaturedDiscount(w http.ResponseWriter, r *http.Request) {
	var req featuredPreviewRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.TargetType != "item" || strings.TrimSpace(req.TargetID) == "" {
		httpjson.Write(w, http.StatusOK, discount.FeaturedConflict{})
		return
	}
	if req.DiscountType != "fixed" && req.DiscountType != "percent" {
		httpjson.Error(w, http.StatusBadRequest, "discount_type must be fixed or percent")
		return
	}
	if req.Amount <= 0 {
		httpjson.Error(w, http.StatusBadRequest, "amount must be > 0")
		return
	}

	var listPrice int
	var categoryID sql.NullString
	err := a.DB.QueryRow(
		`SELECT price_cents, category_id FROM menu_items WHERE id = ?`,
		req.TargetID,
	).Scan(&listPrice, &categoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusBadRequest, "target not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	var catPtr *string
	if categoryID.Valid && categoryID.String != "" {
		catPtr = &categoryID.String
	}
	conflict, err := discount.CheckFeaturedConflict(
		a.DB, req.TargetID, listPrice, catPtr,
		req.DiscountType, req.Amount, strings.TrimSpace(req.ExcludeRuleID), time.Now(),
	)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, http.StatusOK, conflict)
}

func (a *API) writeDiscountRule(w http.ResponseWriter, id string, status int) {
	row := a.DB.QueryRow(`
		SELECT r.id, r.name, r.target_type, r.target_id, r.discount_type, r.amount,
		       r.start_time, r.end_time, r.weekdays, r.starts_on, r.ends_on, r.is_active,
		       COALESCE(r.is_featured_price, 0),
		       r.created_at, r.updated_at,
		       CASE
		         WHEN r.target_type = 'item' THEN (SELECT name FROM menu_items WHERE id = r.target_id)
		         WHEN r.target_type = 'category' THEN (SELECT name FROM menu_categories WHERE id = r.target_id)
		         ELSE NULL
		       END
		FROM discount_rules r WHERE r.id = ?
	`, id)
	rule, err := scanDiscountRule(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpjson.Error(w, http.StatusNotFound, "not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, status, rule)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDiscountRule(s scannable) (discountRuleJSON, error) {
	var r discountRuleJSON
	var start, end, weekdays, startsOn, endsOn sql.NullString
	var active, featured int
	var targetName sql.NullString
	err := s.Scan(
		&r.ID, &r.Name, &r.TargetType, &r.TargetID, &r.DiscountType, &r.Amount,
		&start, &end, &weekdays, &startsOn, &endsOn, &active, &featured,
		&r.CreatedAt, &r.UpdatedAt, &targetName,
	)
	if err != nil {
		return r, err
	}
	r.IsActive = active == 1
	r.IsFeaturedPrice = featured == 1 && r.TargetType == "item"
	r.StartTime = nullToPtr(start)
	r.EndTime = nullToPtr(end)
	r.Weekdays = nullToPtr(weekdays)
	r.StartsOn = nullToPtr(startsOn)
	r.EndsOn = nullToPtr(endsOn)
	if targetName.Valid {
		r.TargetName = targetName.String
	}
	return r, nil
}

func featuredInt(req discountRuleRequest) int {
	if req.TargetType != "item" {
		return 0
	}
	if req.IsFeaturedPrice != nil && *req.IsFeaturedPrice {
		return 1
	}
	return 0
}

func validateDiscountRule(req discountRuleRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name required")
	}
	if req.TargetType != "item" && req.TargetType != "category" {
		return errors.New("target_type must be item or category")
	}
	if strings.TrimSpace(req.TargetID) == "" {
		return errors.New("target_id required")
	}
	if req.DiscountType != "fixed" && req.DiscountType != "percent" {
		return errors.New("discount_type must be fixed or percent")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be > 0")
	}
	if req.DiscountType == "percent" && req.Amount > 100 {
		return errors.New("percent amount must be 1–100")
	}
	if req.IsFeaturedPrice != nil && *req.IsFeaturedPrice && req.TargetType != "item" {
		return errors.New("is_featured_price is only allowed for item rules")
	}
	if err := optionalHHMM(req.StartTime); err != nil {
		return err
	}
	if err := optionalHHMM(req.EndTime); err != nil {
		return err
	}
	if err := optionalDate(req.StartsOn); err != nil {
		return err
	}
	if err := optionalDate(req.EndsOn); err != nil {
		return err
	}
	if req.StartsOn != nil && req.EndsOn != nil &&
		strings.TrimSpace(*req.StartsOn) != "" && strings.TrimSpace(*req.EndsOn) != "" &&
		*req.StartsOn > *req.EndsOn {
		return errors.New("starts_on must be on or before ends_on")
	}
	if req.Weekdays != nil && strings.TrimSpace(*req.Weekdays) != "" {
		for _, part := range strings.Split(*req.Weekdays, ",") {
			part = strings.TrimSpace(part)
			if part < "1" || part > "7" || len(part) != 1 {
				return errors.New("weekdays must be comma-separated 1–7 (Mon–Sun)")
			}
		}
	}
	return nil
}

func optionalHHMM(v *string) error {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	s := strings.TrimSpace(*v)
	if len(s) >= 5 && s[2] == ':' {
		s = s[:5]
	}
	if _, err := time.Parse("15:04", s); err != nil {
		return errors.New("times must be HH:MM")
	}
	*v = s
	return nil
}

func optionalDate(v *string) error {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	s := strings.TrimSpace(*v)
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return errors.New("dates must be YYYY-MM-DD")
	}
	*v = s
	return nil
}

func (a *API) assertDiscountTarget(targetType, targetID string) error {
	var n int
	var q string
	switch targetType {
	case "item":
		q = `SELECT COUNT(*) FROM menu_items WHERE id = ?`
	case "category":
		q = `SELECT COUNT(*) FROM menu_categories WHERE id = ?`
	default:
		return errors.New("invalid target_type")
	}
	if err := a.DB.QueryRow(q, targetID).Scan(&n); err != nil {
		return errors.New("db error")
	}
	if n == 0 {
		return errors.New("target not found")
	}
	return nil
}

func nullStr(v *string) any {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return strings.TrimSpace(*v)
}

func nullToPtr(v sql.NullString) *string {
	if !v.Valid || v.String == "" {
		return nil
	}
	s := v.String
	return &s
}
