package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
)

type categoryJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

func (a *API) ListCategories(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("all") == "1"
	q := `SELECT id, name, sort_order, is_active FROM menu_categories`
	if !includeInactive {
		q += ` WHERE is_active = 1`
	}
	q += ` ORDER BY sort_order, name`

	rows, err := a.DB.Query(q)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	out := []categoryJSON{}
	for rows.Next() {
		var c categoryJSON
		var active int
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder, &active); err != nil {
			httpjson.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		c.IsActive = active == 1
		out = append(out, c)
	}
	httpjson.Write(w, http.StatusOK, out)
}

type categoryRequest struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order"`
}

func (a *API) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpjson.Error(w, http.StatusBadRequest, "name required")
		return
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	id := uuid.NewString()
	_, err := a.DB.Exec(
		`INSERT INTO menu_categories (id, name, sort_order) VALUES (?, ?, ?)`,
		id, name, sortOrder,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			httpjson.Error(w, http.StatusConflict, "category name already exists")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	httpjson.Write(w, http.StatusCreated, categoryJSON{
		ID: id, Name: name, SortOrder: sortOrder, IsActive: true,
	})
}

func (a *API) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req categoryRequest
	if err := httpjson.Decode(r, &req); err != nil {
		httpjson.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpjson.Error(w, http.StatusBadRequest, "name required")
		return
	}

	var sortOrder int
	err := a.DB.QueryRow(`SELECT sort_order FROM menu_categories WHERE id = ? AND is_active = 1`, id).Scan(&sortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	res, err := a.DB.Exec(
		`UPDATE menu_categories SET name = ?, sort_order = ? WHERE id = ? AND is_active = 1`,
		name, sortOrder, id,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			httpjson.Error(w, http.StatusConflict, "category name already exists")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		httpjson.Error(w, http.StatusNotFound, "not found")
		return
	}
	// Keep legacy text column in sync for older rows/UI.
	_, _ = a.DB.Exec(`UPDATE menu_items SET category = ? WHERE category_id = ?`, name, id)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var inUse int
	if err := a.DB.QueryRow(
		`SELECT COUNT(*) FROM menu_items WHERE category_id = ? AND is_active = 1`,
		id,
	).Scan(&inUse); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, "db error")
		return
	}
	if inUse > 0 {
		httpjson.Error(w, http.StatusConflict, "category still used by menu items")
		return
	}

	res, err := a.DB.Exec(
		`UPDATE menu_categories SET is_active = 0 WHERE id = ? AND is_active = 1`,
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

func (a *API) resolveCategoryID(categoryID *string) (*string, *string, error) {
	if categoryID == nil || strings.TrimSpace(*categoryID) == "" {
		return nil, nil, nil
	}
	id := strings.TrimSpace(*categoryID)
	var name string
	err := a.DB.QueryRow(
		`SELECT name FROM menu_categories WHERE id = ? AND is_active = 1`,
		id,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, errors.New("category not found")
	}
	if err != nil {
		return nil, nil, err
	}
	return &id, &name, nil
}
