package server

import (
	"testing"

	"github.com/jayedbinnazir/cart-service/internal/config"
)

// TestRouterSetup ensures every module registers without a Gin route conflict.
// It does not touch Postgres or any downstream service.
func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	cfg.Services.UserServiceURL = "http://user-management:8080"
	cfg.Services.ProductServiceURL = "http://product-service:8080"
	cfg.Services.InventoryServiceURL = "http://inventory-service:8080"

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
