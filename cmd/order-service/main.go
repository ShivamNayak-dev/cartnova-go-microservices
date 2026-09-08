package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/config"
	"github.com/ShivamNayak-dev/cartnova/internal/database"
	"github.com/ShivamNayak-dev/cartnova/internal/events"
	"github.com/ShivamNayak-dev/cartnova/internal/grpcapi"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/ShivamNayak-dev/cartnova/internal/logging"
	"github.com/ShivamNayak-dev/cartnova/internal/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, config.String("DATABASE_URL", "postgres://postgres:root@localhost:5432/cartnova_order?sslmode=disable"))
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool, config.String("MIGRATIONS_DIR", "migrations/order")); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	productConnection, err := grpc.NewClient(config.String("PRODUCT_GRPC_ADDR", "localhost:9092"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("product gRPC connection failed", "error", err)
		os.Exit(1)
	}
	defer productConnection.Close()

	userConnection, err := grpc.NewClient(config.String("USER_GRPC_ADDR", "localhost:9091"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("user gRPC connection failed", "error", err)
		os.Exit(1)
	}
	defer userConnection.Close()

	bus := events.NewBus([]string{config.String("KAFKA_BROKERS", "localhost:29092")})
	defer bus.Close()

	repository := order.NewRepository(pool)
	service := order.NewService(
		repository,
		grpcapi.NewProductServiceClient(productConnection),
		grpcapi.NewUserServiceClient(userConnection),
		bus,
	)
	handler := order.NewHandler(service)
	authService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order-service"})
	})
	mux.Handle("POST /orders", auth.Middleware(authService, http.HandlerFunc(handler.Create)))
	mux.Handle("GET /orders/mine", auth.Middleware(authService, http.HandlerFunc(handler.GetMine)))
	mux.Handle("GET /orders/{id}", auth.Middleware(authService, http.HandlerFunc(handler.GetByID)))
	mux.Handle("PUT /orders/{id}/status", auth.Middleware(authService, http.HandlerFunc(handler.UpdateStatus)))

	consumer := events.NewConsumer([]string{config.String("KAFKA_BROKERS", "localhost:29092")}, order.InventoryEventsTopic, "cartnova-order")
	defer consumer.Close()

	go func() {
		if err := service.ConsumeInventoryEvents(ctx, consumer); err != nil && ctx.Err() == nil {
			logger.Error("order consumer stopped", "error", err)
			stop()
		}
	}()

	server := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8083"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("order service started", "port", server.Addr)
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
