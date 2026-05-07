// movies-api — read-only HTTP catalog API.
//
// Session 1 (tag 0.1.0) ships only /version, /healthz, /readyz.
// The data API (§6) is delivered in later sessions per session-log.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bartr/bartr-movies/internal/config"
	"github.com/bartr/bartr-movies/internal/httpapi"
	"github.com/bartr/bartr-movies/internal/store"
	"github.com/bartr/bartr-movies/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "movies-api: ", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	cfg, err := config.Load(args, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	logger.Info("starting movies-api",
		slog.String("version", version.Version),
		slog.String("config", cfg.Redacted()),
	)

	// Load the in-memory catalog. /readyz stays 503 until this completes,
	// and /api/* responds 503 problem+json until storeRef is non-nil.
	var ready atomic.Bool
	var storeRef atomic.Pointer[store.Store]
	go func() {
		s, err := store.Load(cfg.DataDir)
		if err != nil {
			logger.Error("dataset load failed", slog.String("err", err.Error()))
			return
		}
		c := s.Stats()
		logger.Info("dataset ready",
			slog.Int("movies", c.Movies),
			slog.Int("actors", c.Actors),
			slog.Int("genres", c.Genres),
		)
		storeRef.Store(s)
		ready.Store(true)
	}()

	router := httpapi.NewRouter(
		version.Version,
		func() bool { return ready.Load() },
		func() *store.Store { return storeRef.Load() },
	)

	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(cfg.Port))
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Shutdown plumbing.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("stopped")
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
