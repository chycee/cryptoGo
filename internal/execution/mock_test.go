package execution

import (
	"context"
	"testing"

	"crypto_go/internal/domain"
)

func TestMockExecution_ImplementsInterface(t *testing.T) {
	var _ domain.Execution = (*MockExecution)(nil) // Compile-time check
}

func TestMockExecution_ExecuteOrder(t *testing.T) {
	mock := NewMockExecution()
	order := domain.Order{
		ID:          "test-order-1",
		Symbol:      "BTCUSDT",
		PriceMicros: 100000000,
		QtySats:     10000,
	}

	if err := mock.ExecuteOrder(context.Background(), order); err != nil {
		t.Errorf("ExecuteOrder failed: %v", err)
	}
}

func TestMockExecution_CancelOrder(t *testing.T) {
	mock := NewMockExecution()
	if err := mock.CancelOrder(context.Background(), "test-order-1", "BTCUSDT"); err != nil {
		t.Errorf("CancelOrder failed: %v", err)
	}
}

func TestMockExecution_Close(t *testing.T) {
	mock := NewMockExecution()
	if err := mock.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
