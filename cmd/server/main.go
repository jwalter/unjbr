// Command server runs the unjbr web server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jwalter/unjbr/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	addr := ":" + envOr("PORT", "8080")

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Browsers only allow camera access on secure origins, so serving to
	// other machines needs TLS. localhost works without it.
	certFile, keyFile := os.Getenv("TLS_CERT"), os.Getenv("TLS_KEY")

	errCh := make(chan error, 1)
	go func() {
		var err error
		if certFile != "" && keyFile != "" {
			logger.Info("listening", "addr", addr, "tls", true)
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			logger.Info("listening", "addr", addr)
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
