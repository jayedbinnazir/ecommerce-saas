package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/config"
	"github.com/jayedbinnazir/order-service/internal/events"
	orderpkg "github.com/jayedbinnazir/order-service/internal/order"
	orderservices "github.com/jayedbinnazir/order-service/internal/order/services"
)

type Server struct {
	cfg    *config.Config
	db     *sql.DB
	events *events.Publisher

	// orderSvc is set by registerModules (called from RouterSetup); startConsumer
	// runs after RouterSetup so it's populated by then.
	orderSvc     *orderservices.Service
	consumer     *events.Consumer
	stopConsumer context.CancelFunc
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	return &Server{
		cfg:    cfg,
		db:     db,
		events: events.NewPublisher(cfg.Kafka.Brokers),
	}
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

	s.startConsumer()

	go func() {
		log.Printf("order-service listening on :%s", s.cfg.Server.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}

// startConsumer subscribes to payment-events so a webhook-driven capture or
// failure updates the order even if no client calls Pay again.
func (s *Server) startConsumer() {
	if len(s.cfg.Kafka.Brokers) == 0 {
		log.Println("order-service: no Kafka brokers configured, consumer disabled")
		return
	}
	if s.orderSvc == nil {
		log.Println("order-service: order service not wired, consumer disabled")
		return
	}

	s.consumer = events.NewConsumer(s.cfg.Kafka.Brokers, "order-service", []string{events.TopicPayments})

	ctx, cancel := context.WithCancel(context.Background())
	s.stopConsumer = cancel

	go func() {
		log.Println("order-service consuming payment-events")
		s.consumer.Run(ctx, orderpkg.EventHandler(s.orderSvc))
	}()
}
