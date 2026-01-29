package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true // CORS handled by Gin middleware
	},
}

type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	groupID string
	userID  string
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool // groupID -> clients
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.groupID] == nil {
				h.rooms[client.groupID] = make(map[*Client]bool)
			}
			h.rooms[client.groupID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.groupID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.groupID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastToGroup(groupID string, event WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal WS event: %v", err)
		return
	}

	h.mu.RLock()
	// Copy the client set under lock so we can iterate safely after releasing
	clients := make([]*Client, 0, len(h.rooms[groupID]))
	for c := range h.rooms[groupID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return nil
	}
	return conn
}

func (h *Hub) AddClient(conn *websocket.Conn, groupID, userID string) *Client {
	client := &Client{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, 256),
		groupID: groupID,
		userID:  userID,
	}
	h.register <- client
	go client.writePump()
	go client.readPump()
	return client
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// We don't process incoming messages from clients, WS is server -> client only
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
