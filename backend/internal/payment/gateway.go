package payment

import "context"

type Method string

const (
	MethodCash   Method = "cash"
	MethodCard   Method = "card"
	MethodQR     Method = "qr"
	MethodWallet Method = "wallet"
)

type ChargeRequest struct {
	OrderID     string
	AmountCents int
	Method      Method
}

type ChargeResult struct {
	Provider    string
	ProviderRef string
	Status      string
}

type Gateway interface {
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
}
