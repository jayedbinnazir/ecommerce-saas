package server

import (
	"testing"

	"github.com/redis/go-redis/v9"

	config "github.com/jayedbinnazir/golang-saas.git/internal/config"
)

// TestRouterSetup ensures every module's routes register without a Gin
// wildcard/static conflict panic. It does not touch Postgres or Redis.
func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	cfg.JWT.AccessTTL = "15m"
	cfg.JWT.RefreshTTL = "168h"

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = rdb.Close() })

	s := NewServer(cfg, nil, rdb)

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
