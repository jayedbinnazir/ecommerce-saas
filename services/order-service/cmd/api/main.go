package main

import (
	"log"

	"github.com/jayedbinnazir/order-service/internal/config"
	"github.com/jayedbinnazir/order-service/internal/infrastructure/postgres"
	"github.com/jayedbinnazir/order-service/internal/server"
)

func main() {
	log.Println("starting order-service...")

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

	if err := server.NewServer(cfg, db).Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
