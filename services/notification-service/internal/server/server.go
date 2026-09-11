package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/notification-service/internal/config"
	"github.com/jayedbinnazir/notification-service/internal/events"
	"github.com/jayedbinnazir/notification-service/internal/mailclient"
	"github.com/jayedbinnazir/notification-service/internal/notification"
	"github.com/jayedbinnazir/notification-service/internal/notification/repository"
	"github.com/jayedbinnazir/notification-service/internal/notification/services"
)

type Server struct {
	cfg *config.Config
	db  *sql.DB

	consumer     *events.Consumer
	stopConsumer context.CancelFunc
}

func NewServer(cfg *config.Config, db *sql.DB) *Server {
	return &Server{cfg: cfg, db: db}
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

// Start builds the router, starts the HTTP server and the Kafka consumer, then
// waits for shutdown.
func (s *Server) Start() error {
	router, err := s.RouterSetup()
	if err != nil {
		return err
	}

	httpSrv := s.NewHTTPServer(router)

	s.startConsumer()

	go func() {
		log.Printf("notification-service listening on :%s", s.cfg.Server.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}

// startConsumer subscribes to the order/payment topics and turns each event into
// an in-app notification.
func (s *Server) startConsumer() {
	if len(s.cfg.Kafka.Brokers) == 0 {
		log.Println("notification-service: no Kafka brokers configured, consumer disabled")
		return
	}

	mail := mailclient.New(s.cfg.Services.MailServiceURL, s.cfg.Services.MailInternalKey)
	svc := services.New(repository.New(s.db), mail)

	s.consumer = events.NewConsumer(s.cfg.Kafka.Brokers, "notification-service",
		[]string{events.TopicOrders, events.TopicPayments})

	ctx, cancel := context.WithCancel(context.Background())
	s.stopConsumer = cancel

	go func() {
		log.Println("notification-service consuming order-events, payment-events")
		s.consumer.Run(ctx, notification.EventHandler(svc))
	}()
}
