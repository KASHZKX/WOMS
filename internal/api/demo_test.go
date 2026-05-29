package api

import (
	"testing"

	"github.com/d11nn/woms/internal/auth"
	"github.com/d11nn/woms/internal/domain"
)

func TestNewDemoMemoryStore(t *testing.T) {
	store := NewDemoMemoryStore()
	if store == nil {
		t.Fatal("expected NewDemoMemoryStore to return a MemoryStore, got nil")
	}
	orders := store.ListOrders(auth.Claims{Role: domain.RoleAdmin})
	if len(orders) != 9 {
		t.Errorf("expected 9 seeded orders, got %d", len(orders))
	}
}
