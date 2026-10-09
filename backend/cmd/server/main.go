package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/pos-system-otameshi/backend/internal/auth"
	"github.com/pos-system-otameshi/backend/internal/db"
	"github.com/pos-system-otameshi/backend/internal/handlers"
	authmw "github.com/pos-system-otameshi/backend/internal/middleware"
	"github.com/pos-system-otameshi/backend/internal/payment"
)

func main() {
	addr := env("ADDR", ":8080")
	dbPath := env("DB_PATH", "data/pos.db")
	jwtSecret := env("JWT_SECRET", "dev-secret-change-me")

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := db.Seed(conn); err != nil {
		log.Fatal(err)
	}

	api := &handlers.API{
		DB:      conn,
		Tokens:  auth.NewTokenService(jwtSecret),
		Payment: &payment.MockGateway{},
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Guest-Claim"},
		AllowCredentials: true,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	r.Post("/api/auth/login", api.Login)

	// Phase 4: members (customer accounts)
	r.Post("/api/members/register", api.MemberRegister)
	r.Post("/api/members/login", api.MemberLogin)
	r.Route("/api/member", func(r chi.Router) {
		r.Use(authmw.Auth(api.Tokens))
		r.Use(authmw.RequireRole(auth.RoleMember))
		r.Get("/me", api.MemberMe)
		r.Get("/takeout/active", api.MemberTakeoutActive)
		r.Get("/takeout/history", api.MemberTakeoutHistory)
		r.Get("/takeout/menu", api.TakeoutMenu)
		r.Post("/takeout/orders", api.MemberTakeoutCreate)
		r.Get("/takeout/orders/{orderId}", api.MemberTakeoutGet)
		r.Post("/takeout/orders/{orderId}/items", api.MemberTakeoutAddItem)
		r.Patch("/takeout/orders/{orderId}/items/{itemId}", api.MemberTakeoutUpdateItem)
		r.Post("/takeout/orders/{orderId}/confirm", api.MemberTakeoutConfirm)
		r.Post("/takeout/orders/{orderId}/quit", api.MemberTakeoutQuit)
		r.Post("/takeout/orders/{orderId}/checkout", api.MemberTakeoutCheckout)
	})

	// Phase 2: guest table QR (no staff JWT)
	r.Route("/api/guest/t/{token}", func(r chi.Router) {
		r.Get("/", api.GuestTableStatus)
		r.Get("/menu", api.GuestMenu)
		r.Post("/orders", api.GuestCreateOrder)
		r.Get("/orders/{orderId}", api.GuestGetOrder)
		r.Post("/orders/{orderId}/items", api.GuestAddOrderItem)
		r.Patch("/orders/{orderId}/items/{itemId}", api.GuestUpdateOrderItem)
		r.Post("/orders/{orderId}/confirm", api.GuestConfirmOrderItems)
		r.Post("/orders/{orderId}/quit", api.GuestQuitOrder)
		r.Post("/orders/{orderId}/checkout", api.GuestCheckout)
	})

	// Phase 3: guest takeout (pay-first, cashless)
	r.Route("/api/guest/takeout", func(r chi.Router) {
		r.Get("/menu", api.TakeoutMenu)
		r.Post("/orders", api.TakeoutCreateOrder)
		r.Get("/orders/{orderId}", api.TakeoutGetOrder)
		r.Post("/orders/{orderId}/items", api.TakeoutAddOrderItem)
		r.Patch("/orders/{orderId}/items/{itemId}", api.TakeoutUpdateOrderItem)
		r.Post("/orders/{orderId}/confirm", api.TakeoutConfirmOrderItems)
		r.Post("/orders/{orderId}/quit", api.TakeoutQuitOrder)
		r.Post("/orders/{orderId}/checkout", api.TakeoutCheckout)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(authmw.Auth(api.Tokens))
		r.Use(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter))

		r.Get("/me", api.Me)

		r.Get("/dining-tables", api.ListDiningTables)

		r.Get("/menu", api.ListMenu)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Post("/menu", api.CreateMenuItem)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Put("/menu/{id}", api.UpdateMenuItem)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Delete("/menu/{id}", api.DeleteMenuItem)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Patch("/menu/{id}/sold-out", api.ToggleSoldOut)

		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Get("/categories", api.ListCategories)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Post("/categories", api.CreateCategory)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Put("/categories/{id}", api.UpdateCategory)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Delete("/categories/{id}", api.DeleteCategory)

		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders", api.CreateOrder)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Get("/orders/{id}", api.GetOrder)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/items", api.AddOrderItem)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Patch("/orders/{id}/items/{itemId}", api.UpdateOrderItem)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/confirm", api.ConfirmOrderItems)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/serve", api.MarkOrderServed)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/clear-table", api.ClearTable)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/quit", api.QuitOrder)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/force-cancel", api.ForceCancelOrder)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/orders/{id}/checkout", api.Checkout)

		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Get("/takeout/queue", api.ListTakeoutQueue)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/takeout/orders", api.StaffCreateTakeoutOrder)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/takeout/{id}/ready", api.MarkTakeoutReady)
		r.With(authmw.RequireRole(auth.RoleAdmin, auth.RoleWaiter)).Post("/takeout/{id}/complete", api.MarkTakeoutCompleted)

		r.With(authmw.RequireRole(auth.RoleAdmin)).Get("/reports/transactions", api.ListTransactions)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Get("/reports/summary", api.ReportSummary)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Get("/reports/items", api.ReportItems)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Get("/reports/cancelled", api.ListCancelledOrders)

		r.With(authmw.RequireRole(auth.RoleAdmin)).Get("/discount-rules", api.ListDiscountRules)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Post("/discount-rules", api.CreateDiscountRule)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Post("/discount-rules/preview-featured", api.PreviewFeaturedDiscount)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Put("/discount-rules/{id}", api.UpdateDiscountRule)
		r.With(authmw.RequireRole(auth.RoleAdmin)).Delete("/discount-rules/{id}", api.DeleteDiscountRule)
	})

	log.Printf("API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
