package server

import (
	"testing"

	"github.com/jayedbinnazir/order-service/internal/config"
)

// TestRouterSetup ensures every module registers without a Gin route conflict.
// It does not touch Postgres or any downstream service.
func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	cfg.Services.UserServiceURL = "http://user-management:8080"
	cfg.Services.CartServiceURL = "http://cart-service:8080"
	cfg.Services.InventoryServiceURL = "http://inventory-service:8080"
	cfg.Services.InventoryInternalKey = "test-internal-key"
	cfg.Services.PaymentServiceURL = "http://payment-service:8080"
	cfg.Services.PaymentInternalKey = "test-internal-key"

	s := NewServer(cfg, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("router setup panicked: %v", r)
		}
	}()

	router, err := s.RouterSetup()
	if err != nil {
		t.Fatalf("router setup: %v", err)
	}
	if router == nil {
		t.Fatal("nil router")
	}
}
