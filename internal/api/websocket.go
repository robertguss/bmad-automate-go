package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan WebSocketMessage

	mu     sync.Mutex
	closed bool
}

// WebSocketHub maintains the set of active clients and broadcasts messages
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		stopCh:     make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	h.mu.Lock()
	h.running = true
	h.mu.Unlock()

	for {
		select {
		case <-h.stopCh:
			h.mu.Lock()
			h.running = false
			// Close all clients
			for client := range h.clients {
				client.close()
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, close connection
					go func(c *WebSocketClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop stops the hub
func (h *WebSocketHub) Stop() {
	h.mu.Lock()
	if h.running {
		close(h.stopCh)
	}
	h.mu.Unlock()
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(msg WebSocketMessage) {
	h.mu.RLock()
	running := h.running
	h.mu.RUnlock()

	if running {
		select {
		case h.broadcast <- msg:
		default:
			// Channel full, drop message
		}
	}
}

// ClientCount returns the number of connected clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWs handles WebSocket requests from clients
func (h *WebSocketHub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := &WebSocketClient{
		hub:  h,
		conn: conn,
		send: make(chan WebSocketMessage, 64),
	}

	h.register <- client

	// Start write pump in goroutine
	go client.writePump()

	// Read pump blocks until connection closes
	client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
	}()

	for {
		var msg map[string]interface{}
		err := wsjson.Read(c.conn.CloseRead(nil), c.conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure &&
				websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Handle incoming messages (client commands)
		if msgType, ok := msg["type"].(string); ok {
			c.handleMessage(msgType, msg)
		}
	}
}

// writePump sends messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Channel closed
				return
			}

			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

			ctx, cancel := newWriteContext()
			err := wsjson.Write(ctx, c.conn, message)
			cancel()

			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping
			ctx, cancel := newWriteContext()
			err := c.conn.Ping(ctx)
			cancel()

			if err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming client messages
func (c *WebSocketClient) handleMessage(msgType string, msg map[string]interface{}) {
	switch msgType {
	case "subscribe":
		// Client wants to subscribe to specific event types
		// For now, all clients receive all events
	case "ping":
		// Respond to ping
		c.send <- WebSocketMessage{
			Type:      "pong",
			Timestamp: time.Now(),
		}
	}
}

// close closes the client connection
func (c *WebSocketClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	close(c.send)
	c.conn.Close(websocket.StatusNormalClosure, "closing")
}

// newWriteContext creates a context with timeout for writes
func newWriteContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// Helper types for message data

// ExecutionUpdateData represents execution update data
type ExecutionUpdateData struct {
	StoryKey string  `json:"story_key"`
	Status   string  `json:"status"`
	Step     int     `json:"step"`
	StepName string  `json:"step_name"`
	Progress float64 `json:"progress"`
}

// StepOutputData represents step output data
type StepOutputData struct {
	StoryKey  string `json:"story_key"`
	StepIndex int    `json:"step_index"`
	Line      string `json:"line"`
	IsStderr  bool   `json:"is_stderr"`
}

// QueueUpdateData represents queue update data
type QueueUpdateData struct {
	Total   int    `json:"total"`
	Pending int    `json:"pending"`
	Current int    `json:"current"`
	Status  string `json:"status"`
}

// MarshalJSON implements json.Marshaler for WebSocketMessage
func (m WebSocketMessage) MarshalJSON() ([]byte, error) {
	type Alias WebSocketMessage
	return json.Marshal(&struct {
		Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     Alias(m),
		Timestamp: m.Timestamp.Format(time.RFC3339),
	})
}
