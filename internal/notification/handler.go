package notification

import (
	"net/http"
	"strconv"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
	"github.com/gorilla/websocket"
)

type Handler struct {
	repository *Repository
	hub        *Hub
}

func NewHandler(repository *Repository, hub *Hub) *Handler {
	return &Handler{repository: repository, hub: hub}
}

func (h *Handler) GetMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	notifications, err := h.repository.FindByUserID(r.Context(), claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load notifications")
		return
	}

	httpx.JSON(w, http.StatusOK, notifications)
}

func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(request *http.Request) bool {
			return true
		},
	}

	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	h.userLoop(r, claims.UserID, connection)
}

func (h *Handler) userLoop(r *http.Request, userID int64, connection *websocket.Conn) {
	h.hub.Add(userID, connection)
	defer h.hub.Remove(userID, connection)

	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			return
		}
	}
}

func userIDFromPath(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
