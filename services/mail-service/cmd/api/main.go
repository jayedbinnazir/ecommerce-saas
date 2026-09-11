package main

import (
	"log"

	"github.com/jayedbinnazir/mail-service/internal/config"
	"github.com/jayedbinnazir/mail-service/internal/infrastructure/postgres"
	"github.com/jayedbinnazir/mail-service/internal/server"
	"github.com/jayedbinnazir/mail-service/internal/smtp"
)

func main() {
	log.Println("starting mail-service...")

	cfg, err := config.ConfigLoader()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := postgres.New(cfg.Database)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()
	log.Println("postgres connected")

	sender := smtp.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	if sender.Live() {
		log.Printf("smtp relay: %s", cfg.SMTP.Host)
	} else {
		log.Println("smtp relay: stub (mail is logged, not delivered)")
	}

	if err := server.NewServer(cfg, db, sender).Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
