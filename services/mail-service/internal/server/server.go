package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/mail-service/internal/config"
	"github.com/jayedbinnazir/mail-service/internal/events"
	"github.com/jayedbinnazir/mail-service/internal/mail"
	"github.com/jayedbinnazir/mail-service/internal/mail/repository"
	"github.com/jayedbinnazir/mail-service/internal/mail/services"
	"github.com/jayedbinnazir/mail-service/internal/smtp"
	"github.com/jayedbinnazir/mail-service/internal/userclient"
)

type Server struct {
	cfg    *config.Config
	db     *sql.DB
	sender smtp.Sender

	consumer     *events.Consumer
	stopConsumer context.CancelFunc
}

func NewServer(cfg *config.Config, db *sql.DB, sender smtp.Sender) *Server {
	return &Server{cfg: cfg, db: db, sender: sender}
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

func (s *Server) Start() error {
	router, err := s.RouterSetup()
	if err != nil {
		return err
	}

	httpSrv := s.NewHTTPServer(router)

	s.startConsumer()

	go func() {
		log.Printf("mail-service listening on :%s", s.cfg.Server.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	return s.GracefulShutdown(httpSrv)
}

// startConsumer subscribes to the order/payment topics and sends the matching
// transactional email (resolving the recipient from user-management).
func (s *Server) startConsumer() {
	if len(s.cfg.Kafka.Brokers) == 0 {
		log.Println("mail-service: no Kafka brokers configured, consumer disabled")
		return
	}

	svc := services.New(repository.New(s.db), s.sender)
	users := userclient.New(s.cfg.Services.UserServiceURL, s.cfg.Services.UserInternalKey)

	s.consumer = events.NewConsumer(s.cfg.Kafka.Brokers, "mail-service",
		[]string{events.TopicOrders, events.TopicPayments})

	ctx, cancel := context.WithCancel(context.Background())
	s.stopConsumer = cancel

	go func() {
		log.Println("mail-service consuming order-events, payment-events")
		s.consumer.Run(ctx, mail.EventHandler(svc, users))
	}()
}
