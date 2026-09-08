package notification

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[int64]map[*websocket.Conn]struct{})}
}

func (h *Hub) Add(userID int64, connection *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]struct{})
	}
	h.clients[userID][connection] = struct{}{}
}

func (h *Hub) Remove(userID int64, connection *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients := h.clients[userID]; clients != nil {
		delete(clients, connection)
		if len(clients) == 0 {
			delete(h.clients, userID)
		}
	}
}

func (h *Hub) Broadcast(userID int64, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients[userID]))
	for connection := range h.clients[userID] {
		clients = append(clients, connection)
	}
	h.mu.RUnlock()

	for _, connection := range clients {
		if err := connection.WriteMessage(websocket.TextMessage, data); err != nil {
			h.Remove(userID, connection)
			_ = connection.Close()
		}
	}
}
