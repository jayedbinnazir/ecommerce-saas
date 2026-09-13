package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	config "github.com/jayedbinnazir/golang-saas.git/internal/config"
)

type Server struct {
	cfg *config.Config
	db  *sql.DB
	rdb *redis.Client
}

func NewServer(cfg *config.Config, db *sql.DB, rdb *redis.Client) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
		rdb: rdb,
	}
}

// NewHTTPServer creates and configures the HTTP server.
func (s *Server) NewHTTPServer(router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", s.cfg.UserService.Port),
		Handler: router,

		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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
		log.Printf("Server is running on port %s", s.cfg.UserService.Port)

		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}
