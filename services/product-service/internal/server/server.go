package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/config"
	objstore "github.com/jayedbinnazir/product-service/internal/infrastructure/s3"
)

type Server struct {
	cfg     *config.Config
	db      *sql.DB
	storage *objstore.Storage
}

func NewServer(cfg *config.Config, db *sql.DB, storage *objstore.Storage) *Server {
	return &Server{cfg: cfg, db: db, storage: storage}
}

func (s *Server) NewHTTPServer(router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:           fmt.Sprintf(":%s", s.cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

// Start builds the router, starts the HTTP server and waits for shutdown.
func (s *Server) Start() error {
	router, err := s.RouterSetup()
	if err != nil {
		return err
	}

	httpSrv := s.NewHTTPServer(router)

	go func() {
		log.Printf("product-service listening on :%s", s.cfg.Server.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}
