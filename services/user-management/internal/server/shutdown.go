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

// GracefulShutdown waits for an operating system signal
// and gracefully shuts down the HTTP server.
func (s *Server) GracefulShutdown(httpSrv *http.Server) error {

	// Create a channel for receiving OS signals.
	quit := make(chan os.Signal, 1)

	// Listen for:
	// SIGINT  -> Ctrl + C
	// SIGTERM -> termination request
	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	// Block here until a signal is received.
	<-quit

	log.Println("Shutting down server...")

	// Give active requests 5 seconds to finish.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	// Gracefully shut down the HTTP server.
	if err := httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf(
			"server shutdown failed: %w",
			err,
		)
	}

	log.Println("Server shutdown completed")

	return nil
}
