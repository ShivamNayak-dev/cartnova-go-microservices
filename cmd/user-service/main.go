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
	"github.com/ShivamNayak-dev/cartnova/internal/user"
	"google.golang.org/grpc"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := config.String("DATABASE_URL", "postgres://postgres:root@localhost:5432/cartnova_user?sslmode=disable")
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrationDir := config.String("MIGRATIONS_DIR", "migrations/user")
	if err := database.RunMigrations(ctx, pool, migrationDir); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	authService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))
	repository := user.NewRepository(pool)
	service := user.NewService(repository)
	handler := user.NewHandler(service, authService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "user-service"})
	})
	mux.HandleFunc("POST /register", handler.Register)
	mux.HandleFunc("POST /login", handler.Login)
	mux.Handle("GET /users/{id}", auth.Middleware(authService, http.HandlerFunc(handler.GetByID)))
	mux.Handle("PUT /users/{id}", auth.Middleware(authService, http.HandlerFunc(handler.UpdateProfile)))

	httpServer := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8081"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	grpcapi.RegisterUserServiceServer(grpcServer, user.NewGRPCServer(service))

	grpcListener, err := net.Listen("tcp", ":"+config.String("GRPC_PORT", "9091"))
	if err != nil {
		logger.Error("gRPC listener failed", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("user service HTTP started", "port", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server stopped", "error", err)
			stop()
		}
	}()

	go func() {
		logger.Info("user service gRPC started", "port", grpcListener.Addr().String())
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
