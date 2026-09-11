package server

import (
	"testing"

	"github.com/jayedbinnazir/notification-service/internal/config"
)

func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	cfg.Services.UserServiceURL = "http://user-management:8080"
	cfg.InternalKey = "test-internal-key"

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
