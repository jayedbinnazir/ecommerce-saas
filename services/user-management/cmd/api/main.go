package main

import (
	"log"

	config "github.com/jayedbinnazir/golang-saas.git/internal/config"
	"github.com/jayedbinnazir/golang-saas.git/internal/infrastructure/postgres"
	redisinfra "github.com/jayedbinnazir/golang-saas.git/internal/infrastructure/redis"
	"github.com/jayedbinnazir/golang-saas.git/internal/server"
)

func main() {
	log.Println("Starting user-management service...")

	cfg, err := config.ConfigLoader()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := postgres.New(cfg.Database)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()
	log.Println("PostgreSQL connected")

	rdb, err := redisinfra.New(cfg.Redis)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()
	log.Println("Redis connected")

	srv := server.NewServer(cfg, db, rdb)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
