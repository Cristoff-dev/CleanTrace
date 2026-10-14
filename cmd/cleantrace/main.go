package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cristoff-dev/cleantrace/internal/ingestion"
	"github.com/Cristoff-dev/cleantrace/internal/sanitizer"
	"github.com/Cristoff-dev/cleantrace/internal/storage"
	"github.com/Cristoff-dev/cleantrace/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	log := logger.New(env)
	defer func() {
		_ = log.Sync()
	}()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	maskerEngine := sanitizer.NewRegexMasker()
	postgresRepo := storage.NewPostgresRepo(dbPool)
	httpHandler := ingestion.NewHTTPHandler(maskerEngine, postgresRepo)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "healthy"}`))
	})

	mux.HandleFunc("POST /api/v1/logs/collect", httpHandler.CollectLogs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("Starting CleanTrace server", zap.String("port", port), zap.String("env", env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server stopped cleanly")
}
