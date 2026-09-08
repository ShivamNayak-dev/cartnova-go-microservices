package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/config"
	"github.com/ShivamNayak-dev/cartnova/internal/database"
	"github.com/ShivamNayak-dev/cartnova/internal/grpcapi"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/ShivamNayak-dev/cartnova/internal/logging"
	"github.com/ShivamNayak-dev/cartnova/internal/product"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := config.String("DATABASE_URL", "postgres://postgres:root@localhost:5432/cartnova_product?sslmode=disable")
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrationDir := config.String("MIGRATIONS_DIR", "migrations/product")
	if err := database.RunMigrations(ctx, pool, migrationDir); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: config.String("REDIS_ADDR", "localhost:6379"),
	})
	defer redisClient.Close()

	repository := product.NewRepository(pool)
	service := product.NewService(repository, redisClient)
	handler := product.NewHandler(service)
	categoryRepository := product.NewCategoryRepository(pool)
	categoryHandler := product.NewCategoryHandler(categoryRepository)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "product-service"})
	})
	authService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))
	mux.Handle("POST /products", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(handler.Create)))
	mux.HandleFunc("GET /products", handler.GetAll)
	mux.HandleFunc("GET /products/{id}", handler.GetByID)
	mux.Handle("PUT /products/{id}", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(handler.Update)))
	mux.Handle("DELETE /products/{id}", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(handler.Delete)))
	mux.Handle("POST /categories", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(categoryHandler.Create)))
	mux.Handle("GET /categories", categoryHandler.GetAll)
	mux.Handle("GET /categories/{id}", categoryHandler.GetByID)
	mux.Handle("PUT /categories/{id}", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(categoryHandler.Update)))
	mux.Handle("DELETE /categories/{id}", auth.RequireRole(authService, "ADMIN", http.HandlerFunc(categoryHandler.Delete)))

	httpServer := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8082"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	grpcapi.RegisterProductServiceServer(grpcServer, product.NewGRPCServer(service))
	grpcListener, err := net.Listen("tcp", ":"+config.String("GRPC_PORT", "9092"))
	if err != nil {
		logger.Error("gRPC listener failed", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("product service HTTP started", "port", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server stopped", "error", err)
			stop()
		}
	}()

	go func() {
		logger.Info("product service gRPC started", "port", grpcListener.Addr().String())
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("gRPC server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()
}
