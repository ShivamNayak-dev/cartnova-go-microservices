package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/config"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/ShivamNayak-dev/cartnova/internal/logging"
	"github.com/gorilla/websocket"
)

func main() {
	logger := logging.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jwtService := auth.NewService(config.String("JWT_SECRET", "change-me-in-production"))

	userProxy := newProxy(config.String("USER_SERVICE_URL", "http://localhost:8081"), "/api/v1/auth")
	userAPIProxy := newProxy(config.String("USER_SERVICE_URL", "http://localhost:8081"), "/api/v1")
	productProxy := newProxy(config.String("PRODUCT_SERVICE_URL", "http://localhost:8082"), "/api/v1")
	orderProxy := newProxy(config.String("ORDER_SERVICE_URL", "http://localhost:8083"), "/api/v1")
	inventoryProxy := newProxy(config.String("INVENTORY_SERVICE_URL", "http://localhost:8084"), "/api/v1")
	notificationProxy := newProxy(config.String("NOTIFICATION_SERVICE_URL", "http://localhost:8085"), "/api/v1")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-gateway"})
	})

	mux.Handle("POST /api/v1/auth/register", userProxy)
	mux.Handle("POST /api/v1/auth/login", userProxy)

	protected := func(handler http.Handler) http.Handler {
		return auth.Middleware(jwtService, handler)
	}

	mux.Handle("GET /api/v1/users/{id}", protected(userAPIProxy))
	mux.Handle("PUT /api/v1/users/{id}", protected(userAPIProxy))

	mux.Handle("GET /api/v1/products", productProxy)
	mux.Handle("GET /api/v1/products/{id}", productProxy)
	mux.Handle("POST /api/v1/products", protected(productProxy))
	mux.Handle("PUT /api/v1/products/{id}", protected(productProxy))
	mux.Handle("DELETE /api/v1/products/{id}", protected(productProxy))
	mux.Handle("POST /api/v1/categories", protected(productProxy))
	mux.Handle("GET /api/v1/categories", productProxy)
	mux.Handle("GET /api/v1/categories/{id}", productProxy)
	mux.Handle("PUT /api/v1/categories/{id}", protected(productProxy))
	mux.Handle("DELETE /api/v1/categories/{id}", protected(productProxy))

	mux.Handle("POST /api/v1/orders", protected(orderProxy))
	mux.Handle("GET /api/v1/orders/mine", protected(orderProxy))
	mux.Handle("GET /api/v1/orders/{id}", protected(orderProxy))
	mux.Handle("PUT /api/v1/orders/{id}/status", protected(orderProxy))

	mux.Handle("GET /api/v1/inventory/{productID}", protected(inventoryProxy))
	mux.Handle("POST /api/v1/inventory", auth.RequireRole(jwtService, "ADMIN", inventoryProxy))

	mux.Handle("GET /api/v1/notifications", protected(notificationProxy))
	mux.Handle("GET /ws/notifications", protected(websocketProxy(config.String("NOTIFICATION_SERVICE_URL", "http://localhost:8085"))))

	server := &http.Server{
		Addr:              ":" + config.String("HTTP_PORT", "8080"),
		Handler:           corsMiddleware(loggingMiddleware(logger, mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("API gateway started", "port", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("gateway stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func newProxy(rawTarget string, stripPrefix string) http.Handler {
	target, err := url.Parse(rawTarget)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		originalDirector(request)
		request.URL.Path = strings.TrimPrefix(request.URL.Path, stripPrefix)
		if request.URL.Path == "" {
			request.URL.Path = "/"
		}
		request.Header.Set("X-Forwarded-Host", request.Host)
	}

	return proxy
}

func websocketProxy(rawTarget string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := strings.Replace(rawTarget, "http://", "ws://", 1)
		target = strings.Replace(target, "https://", "wss://", 1)
		target += "/ws"
		if token := r.URL.Query().Get("token"); token != "" {
			target += "?token=" + url.QueryEscape(token)
		}

		header := http.Header{}
		for key, values := range r.Header {
			header[key] = values
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(request *http.Request) bool {
				return true
			},
		}

		clientConnection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer clientConnection.Close()

		serverConnection, response, err := websocket.DefaultDialer.Dial(target, header)
		if err != nil {
			if response != nil {
				_ = response.Body.Close()
			}
			return
		}
		defer serverConnection.Close()

		errorChannel := make(chan error, 2)

		go func() {
			for {
				messageType, data, err := clientConnection.ReadMessage()
				if err != nil {
					errorChannel <- err
					return
				}
				if err := serverConnection.WriteMessage(messageType, data); err != nil {
					errorChannel <- err
					return
				}
			}
		}()

		go func() {
			for {
				messageType, data, err := serverConnection.ReadMessage()
				if err != nil {
					errorChannel <- err
					return
				}
				if err := clientConnection.WriteMessage(messageType, data); err != nil {
					errorChannel <- err
					return
				}
			}
		}()

		<-errorChannel

	})
}

func loggingMiddleware(logger interface {
	Info(string, ...any)
	Error(string, ...any)
}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
