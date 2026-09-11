package main

import (
	"context"
	"log"

	"github.com/jayedbinnazir/product-service/internal/config"
	"github.com/jayedbinnazir/product-service/internal/infrastructure/postgres"
	objstore "github.com/jayedbinnazir/product-service/internal/infrastructure/s3"
	"github.com/jayedbinnazir/product-service/internal/server"
)

func main() {
	log.Println("starting product-service...")

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

	storage, err := objstore.New(context.Background(), cfg.S3)
	if err != nil {
		log.Fatalf("init object storage: %v", err)
	}
	log.Printf("object storage ready (bucket %q)", cfg.S3.Bucket)

	if err := server.NewServer(cfg, db, storage).Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
