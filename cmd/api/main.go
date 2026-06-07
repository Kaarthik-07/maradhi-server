package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/maradhi-api/internal/config"
	"github.com/yourusername/maradhi-api/internal/handler"
	"github.com/yourusername/maradhi-api/internal/middleware"
)

func main() {
	loadDotEnv(".env")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if os.Getenv("ENV") == "production" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config failed", "error", err)
		os.Exit(1)
	}

	db, err := newPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("db failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("db connected")

	mux := http.NewServeMux()
	h := handler.New(db, cfg)
	h.RegisterRoutes(mux)

	app := middleware.Chain(mux,
		middleware.Logger,
		middleware.Recovery,
		middleware.CORS(cfg.AllowedOrigins),
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	slog.Info("done")
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if idx := strings.Index(value, " #"); idx != -1 {
			value = strings.TrimSpace(value[:idx])
		}
		value = strings.Trim(value, `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func newPool(url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
