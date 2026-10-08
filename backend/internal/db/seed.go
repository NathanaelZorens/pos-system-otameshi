package db

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func Seed(conn *sql.DB) error {
	if err := ensureDemoMember(conn); err != nil {
		return err
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	waiterHash, err := bcrypt.GenerateFromPassword([]byte("waiter123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminID := uuid.NewString()
	waiterID := uuid.NewString()

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)`,
		adminID, "admin@pos.local", string(adminHash), "Admin", "admin",
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)`,
		waiterID, "waiter@pos.local", string(waiterHash), "Waiter", "waiter",
	); err != nil {
		return err
	}

	for i := 1; i <= 8; i++ {
		label := fmt.Sprintf("%d", i)
		if _, err := tx.Exec(
			`INSERT INTO dining_tables (id, label, capacity, guest_token) VALUES (?, ?, ?, ?)`,
			uuid.NewString(), label, 4, uuid.NewString(),
		); err != nil {
			return err
		}
	}

	catIDs := map[string]string{}
	for i, name := range []string{
		"メイン", "定食", "スープ",
		"Boulangerie", "Entrées", "Plats", "Desserts",
		"Antipasti", "Primi", "Pizze", "Dolci",
	} {
		id := uuid.NewString()
		catIDs[name] = id
		if _, err := tx.Exec(
			`INSERT INTO menu_categories (id, name, sort_order) VALUES (?, ?, ?)`,
			id, name, i,
		); err != nil {
			return err
		}
	}

	menu := []struct {
		name     string
		yen      int
		category string
	}{
		{"チキンカツ", 980, "メイン"},
		{"カレー定食", 900, "定食"},
		{"ハンバーガー", 850, "メイン"},
		{"チキンライス", 800, "メイン"},
		{"チキンスープ", 450, "スープ"},
		{"Croissant", 320, "Boulangerie"},
		{"Soupe à l'oignon", 680, "Entrées"},
		{"Quiche Lorraine", 920, "Plats"},
		{"Coq au vin", 1580, "Plats"},
		{"Crème brûlée", 650, "Desserts"},
		{"Bruschetta", 580, "Antipasti"},
		{"Spaghetti Carbonara", 1280, "Primi"},
		{"Pizza Margherita", 1350, "Pizze"},
		{"Risotto ai funghi", 1420, "Primi"},
		{"Tiramisù", 720, "Dolci"},
	}
	for _, m := range menu {
		cid := catIDs[m.category]
		if _, err := tx.Exec(
			`INSERT INTO menu_items (id, name, price_cents, category, category_id) VALUES (?, ?, ?, ?, ?)`,
			uuid.NewString(), m.name, m.yen, m.category, cid,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ensureDemoMember(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM members WHERE email = ?`, "member@pos.local").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("member123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		`INSERT INTO members (id, email, password_hash, name) VALUES (?, ?, ?, ?)`,
		uuid.NewString(), "member@pos.local", string(hash), "Demo Member",
	)
	return err
}
