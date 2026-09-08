package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/config"
	"github.com/ShivamNayak-dev/cartnova/internal/database"
	"github.com/ShivamNayak-dev/cartnova/internal/events"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/ShivamNayak-dev/cartnova/internal/inventory"
	"github.com/ShivamNayak-dev/cartnova/internal/logging"
	"net/http"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, config.String("DATABASE_URL", "postgres://postgres:root@localhost:5432/cartnova_inventory?sslmode=disable"))
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool, config.String("MIGRATIONS_DIR", "migrations/inventory")); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	brokers := []string{config.String("KAFKA_BROKERS", "localhost:29092")}
	bus := events.NewBus(brokers)
	defer bus.Close()

	repository := inventory.NewRepository(pool)
	service := inventory.NewService(repository, bus, config.Int("WORKER_COUNT", 4))
	handler := inventory.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "inventory-service"})
	})

	authService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))
	mux.Handle("POST /inventory", auth.Middleware(authService, http.HandlerFunc(handler.AddStock)))
	mux.Handle("GET /inventory/{productID}", auth.Middleware(authService, http.HandlerFunc(handler.Get)))

	consumer := events.NewConsumer(brokers, inventory.OrderEventsTopic, "cartnova-inventory")
	defer consumer.Close()

	go func() {
		if err := service.ConsumeOrders(ctx, consumer); err != nil && ctx.Err() == nil {
			logger.Error("inventory consumer stopped", "error", err)
			stop()
		}
	}()

	server := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8084"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("inventory service started", "port", server.Addr, "workers", config.Int("WORKER_COUNT", 4))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
