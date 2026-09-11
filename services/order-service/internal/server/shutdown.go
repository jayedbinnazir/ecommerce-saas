package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulShutdown blocks until SIGINT/SIGTERM, then drains the HTTP server.
func (s *Server) GracefulShutdown(httpSrv *http.Server) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	if s.stopConsumer != nil {
		s.stopConsumer()
		if err := s.consumer.Close(); err != nil {
			log.Printf("consumer close: %v", err)
		}
	}
	if err := s.events.Close(); err != nil {
		log.Printf("events publisher close: %v", err)
	}
	log.Println("shutdown complete")
	return nil
}
