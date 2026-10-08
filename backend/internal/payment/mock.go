package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type MockGateway struct {
	FailNext bool
}

func (m *MockGateway) Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
	if m.FailNext {
		m.FailNext = false
		return nil, fmt.Errorf("mock payment declined")
	}
	return &ChargeResult{
		Provider:    "mock",
		ProviderRef: uuid.NewString(),
		Status:      "succeeded",
	}, nil
}
