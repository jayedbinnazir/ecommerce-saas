package server

import (
	"testing"

	"github.com/jayedbinnazir/mail-service/internal/config"
	"github.com/jayedbinnazir/mail-service/internal/smtp"
)

func TestRouterSetup(t *testing.T) {
	cfg := &config.Config{}
	cfg.InternalKey = "test-internal-key"

	s := NewServer(cfg, nil, smtp.New("", "", "", "", ""))

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
