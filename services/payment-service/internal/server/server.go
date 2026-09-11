package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/payment-service/internal/config"
	"github.com/jayedbinnazir/payment-service/internal/events"
)

type Server struct {
	cfg    *config.Config
	db     *sql.DB
	events *events.Publisher
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	return &Server{cfg: cfg, db: db, events: events.NewPublisher(cfg.Kafka.Brokers)}
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
		log.Printf("payment-service listening on :%s", s.cfg.Server.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}
