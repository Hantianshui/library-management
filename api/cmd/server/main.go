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

	"library-management/api/internal/httpapi"
	"library-management/api/internal/library"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := env("DATABASE_URL", "postgres://library:library@localhost:5432/library?sslmode=disable")
	jwtSecret := env("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		logger.Error("JWT_SECRET must contain at least 32 characters")
		os.Exit(1)
	}
	adminUser := env("ADMIN_USERNAME", "admin")
	adminPassword := env("ADMIN_PASSWORD", "")
	if len(adminPassword) < 8 {
		logger.Error("ADMIN_PASSWORD must contain at least 8 characters")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := library.NewStore(ctx, databaseURL)
	cancel()
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := store.Migrate(ctx, env("MIGRATIONS_DIR", "./migrations")); err != nil {
		cancel()
		logger.Error("run database migrations", "error", err)
		os.Exit(1)
	}
	if err := store.EnsureAdmin(ctx, adminUser, adminPassword); err != nil {
		cancel()
		logger.Error("create initial admin", "error", err)
		os.Exit(1)
	}
	cancel()

	server := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           httpapi.New(store, jwtSecret, env("COOKIE_SECURE", "false") == "true", logger).Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("api started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("api stopped")
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
