package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*Client]struct{}
}

type Client struct {
	hub    *Hub
	userID int64
	conn   *websocket.Conn
	send   chan []byte
}

type wsMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[*Client]struct{}),
	}
}

func (h *Hub) Register(userID int64, conn *websocket.Conn) {
	client := &Client{
		hub:    h,
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 16),
	}

	h.mu.Lock()
	if _, ok := h.clients[userID]; !ok {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][client] = struct{}{}
	h.mu.Unlock()

	go client.writePump()
	go client.readPump()
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userClients, ok := h.clients[client.userID]
	if !ok {
		return
	}

	if _, exists := userClients[client]; exists {
		delete(userClients, client)
		close(client.send)
	}

	if len(userClients) == 0 {
		delete(h.clients, client.userID)
	}
}

func (h *Hub) Broadcast(eventType string, payload interface{}) {
	message, err := json.Marshal(wsMessage{Type: eventType, Payload: payload})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userClients := range h.clients {
		for client := range userClients {
			select {
			case client.send <- message:
			default:
				go h.unregister(client)
			}
		}
	}
}

func (h *Hub) SendToUser(userID int64, eventType string, payload interface{}) {
	message, err := json.Marshal(wsMessage{Type: eventType, Payload: payload})
	if err != nil {
		return
	}

	h.mu.RLock()
	userClients := h.clients[userID]
	h.mu.RUnlock()

	for client := range userClients {
		select {
		case client.send <- message:
		default:
			go h.unregister(client)
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		_ = c.conn.Close()
	}()

	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
