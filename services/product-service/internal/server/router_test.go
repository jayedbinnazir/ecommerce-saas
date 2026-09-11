package server

import (
	"testing"

	"github.com/jayedbinnazir/product-service/internal/config"
)

// TestRouterSetup ensures every module registers without a Gin route conflict.
// It does not touch Postgres or S3.
func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	cfg.Services.UserServiceURL = "http://user-management:8080"

	s := NewServer(cfg, nil, nil)

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
