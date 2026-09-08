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
	"github.com/ShivamNayak-dev/cartnova/internal/events"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/ShivamNayak-dev/cartnova/internal/logging"
	"github.com/ShivamNayak-dev/cartnova/internal/notification"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.String("MONGODB_URL", "mongodb://root:root@localhost:27017/?authSource=admin")))
	if err != nil {
		logger.Error("MongoDB connection failed", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(ctx, nil); err != nil {
		logger.Error("MongoDB ping failed", "error", err)
		os.Exit(1)
	}

	databaseName := config.String("MONGODB_DATABASE", "cartnova_notification")
	repository := notification.NewRepository(mongoClient.Database(databaseName))
	hub := notification.NewHub()
	service := notification.NewService(repository, hub)
	handler := notification.NewHandler(repository, hub)

	authService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "notification-service"})
	})
	mux.Handle("GET /notifications", auth.Middleware(authService, http.HandlerFunc(handler.GetMine)))
	mux.Handle("GET /ws", auth.Middleware(authService, http.HandlerFunc(handler.WebSocket)))

	brokers := []string{config.String("KAFKA_BROKERS", "localhost:29092")}
	consumer := events.NewConsumer(brokers, "order-events", "cartnova-notification")
	defer consumer.Close()

	go func() {
		if err := service.ConsumeOrderEvents(ctx, consumer); err != nil && ctx.Err() == nil {
			logger.Error("notification consumer stopped", "error", err)
			stop()
		}
	}()

	server := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8085"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("notification service started", "port", server.Addr)
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
