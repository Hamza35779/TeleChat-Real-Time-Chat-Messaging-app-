package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

type Client struct {
	ID       string          `json:"id"`
	Username string          `json:"username"`
	Conn     *websocket.Conn `json:"-"`
	Send     chan []byte     `json:"-"`
	IsTyping bool            `json:"isTyping"`
	LastSeen time.Time       `json:"lastSeen"`
}

type Message struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Username  string    `json:"username"`
	UserID    string    `json:"userId"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Edited    bool      `json:"edited"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	messages   []Message
	mutex      sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		messages:   make([]Message, 0),
	}
}

func (h *Hub) run() {
	log.Println("🔄 Hub started and running...")
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mutex.Unlock()

			log.Printf("➕ Client %s (%s) connected. Total clients: %d", client.Username, client.ID, clientCount)

			// Send recent messages to new client
			h.sendRecentMessages(client)

			// Broadcast user joined - give a small delay to ensure connection is ready
			go func() {
				time.Sleep(100 * time.Millisecond)
				log.Printf("🔄 Broadcasting user list after client %s joined", client.Username)
				h.broadcastUserList()
			}()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			clientCount := len(h.clients)
			h.mutex.Unlock()

			log.Printf("➖ Client %s (%s) disconnected. Total clients: %d", client.Username, client.ID, clientCount)

			// Broadcast user left
			h.broadcastUserList()

		case message := <-h.broadcast:
			h.mutex.RLock()
			clientCount := len(h.clients)
			log.Printf("📢 Broadcasting message to %d clients: %s", clientCount, string(message))

			successCount := 0
			failedClients := make([]*Client, 0)

			for client := range h.clients {
				select {
				case client.Send <- message:
					successCount++
					log.Printf("✅ Sent message to %s", client.Username)
				default:
					log.Printf("❌ Failed to send to client %s, marking for removal", client.Username)
					failedClients = append(failedClients, client)
				}
			}
			h.mutex.RUnlock()

			// Clean up failed clients
			if len(failedClients) > 0 {
				h.mutex.Lock()
				for _, client := range failedClients {
					if _, ok := h.clients[client]; ok {
						close(client.Send)
						delete(h.clients, client)
					}
				}
				h.mutex.Unlock()
				log.Printf("🧹 Cleaned up %d failed clients", len(failedClients))
			}

			log.Printf("✅ Message sent to %d/%d clients", successCount, clientCount)
		}
	}
}

func (h *Hub) sendRecentMessages(client *Client) {
	h.mutex.RLock()
	recentMessages := h.messages
	if len(recentMessages) > 50 {
		recentMessages = recentMessages[len(recentMessages)-50:]
	}
	messageCount := len(recentMessages)
	h.mutex.RUnlock()

	log.Printf("📜 Sending %d recent messages to %s", messageCount, client.Username)

	for _, msg := range recentMessages {
		msgBytes, _ := json.Marshal(msg)
		select {
		case client.Send <- msgBytes:
		default:
			log.Printf("❌ Failed to send recent message to %s", client.Username)
			// Don't close or delete client here - let the writePump handle it
			return
		}
	}
}

func (h *Hub) broadcastUserList() {
	h.mutex.RLock()
	users := make([]Client, 0, len(h.clients))
	for client := range h.clients {
		users = append(users, Client{
			ID:       client.ID,
			Username: client.Username,
			IsTyping: client.IsTyping,
			LastSeen: client.LastSeen,
		})
	}
	userCount := len(users)
	h.mutex.RUnlock()

	log.Printf("👥 Broadcasting user list: %d users", userCount)
	for _, user := range users {
		log.Printf("   - %s (typing: %v)", user.Username, user.IsTyping)
	}

	userListMsg := map[string]interface{}{
		"type":      "userList",
		"users":     users,
		"count":     userCount,
		"timestamp": time.Now(),
	}

	msgBytes, _ := json.Marshal(userListMsg)
	h.broadcast <- msgBytes
}

func (h *Hub) addMessage(msg Message) {
	h.mutex.Lock()
	h.messages = append(h.messages, msg)
	messageCount := len(h.messages)
	h.mutex.Unlock()
	log.Printf("💾 Message stored. Total messages: %d", messageCount)
}

func (h *Hub) editMessage(messageID, userID, newContent string) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for i, msg := range h.messages {
		if msg.ID == messageID && msg.UserID == userID {
			log.Printf("✏️ Editing message %s by %s", messageID, userID)
			h.messages[i].Content = newContent
			h.messages[i].Edited = true
			return true
		}
	}
	log.Printf("❌ Message %s not found for editing by %s", messageID, userID)
	return false
}

func (h *Hub) deleteMessage(messageID, userID string) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for i, msg := range h.messages {
		if msg.ID == messageID && msg.UserID == userID {
			log.Printf("🗑️ Deleting message %s by %s", messageID, userID)
			h.messages = append(h.messages[:i], h.messages[i+1:]...)
			return true
		}
	}
	log.Printf("❌ Message %s not found for deletion by %s", messageID, userID)
	return false
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		log.Printf("🔌 Closing connection for %s", c.Username)
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	log.Printf("📖 Starting read pump for %s", c.Username)

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Unexpected close error for %s: %v", c.Username, err)
			} else {
				log.Printf("🔌 Connection closed for %s: %v", c.Username, err)
			}
			break
		}

		log.Printf("📨 Received raw message from %s: %s", c.Username, string(messageBytes))

		var incomingMsg map[string]interface{}
		if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
			log.Printf("❌ JSON unmarshal error from %s: %v", c.Username, err)
			continue
		}

		msgType, ok := incomingMsg["type"].(string)
		if !ok {
			log.Printf("❌ No message type from %s", c.Username)
			continue
		}

		log.Printf("📥 Processing message type '%s' from %s", msgType, c.Username)

		switch msgType {
		case "message":
			content, ok := incomingMsg["content"].(string)
			if !ok || content == "" {
				log.Printf("❌ Invalid message content from %s", c.Username)
				continue
			}

			message := Message{
				ID:        uuid.New().String(),
				Type:      "message",
				Username:  c.Username,
				UserID:    c.ID,
				Content:   content,
				Timestamp: time.Now(),
				Edited:    false,
			}

			log.Printf("💬 New message from %s: %s", c.Username, content)
			hub.addMessage(message)
			msgBytes, _ := json.Marshal(message)
			hub.broadcast <- msgBytes

		case "typing":
			isTyping, ok := incomingMsg["isTyping"].(bool)
			if !ok {
				log.Printf("❌ Invalid typing status from %s", c.Username)
				continue
			}
			log.Printf("⌨️ %s typing status: %v", c.Username, isTyping)
			c.IsTyping = isTyping
			hub.broadcastUserList()

		case "edit":
			messageID, ok1 := incomingMsg["messageId"].(string)
			newContent, ok2 := incomingMsg["content"].(string)
			if !ok1 || !ok2 {
				log.Printf("❌ Invalid edit request from %s", c.Username)
				continue
			}

			if hub.editMessage(messageID, c.ID, newContent) {
				editMsg := map[string]interface{}{
					"type":      "messageEdited",
					"messageId": messageID,
					"content":   newContent,
					"timestamp": time.Now(),
				}
				msgBytes, _ := json.Marshal(editMsg)
				hub.broadcast <- msgBytes
			}

		case "delete":
			messageID, ok := incomingMsg["messageId"].(string)
			if !ok {
				log.Printf("❌ Invalid delete request from %s", c.Username)
				continue
			}

			if hub.deleteMessage(messageID, c.ID) {
				deleteMsg := map[string]interface{}{
					"type":      "messageDeleted",
					"messageId": messageID,
					"timestamp": time.Now(),
				}
				msgBytes, _ := json.Marshal(deleteMsg)
				hub.broadcast <- msgBytes
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		log.Printf("📝 Write pump stopped for %s", c.Username)
	}()

	log.Printf("📝 Starting write pump for %s", c.Username)

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				log.Printf("📤 Send channel closed for %s", c.Username)
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Write message error for %s: %v", c.Username, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("❌ Ping error for %s: %v", c.Username, err)
				return
			}
		}
	}
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	log.Printf("🔗 WebSocket connection request from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Anonymous"
	}

	log.Printf("👤 Creating client for username: %s", username)

	client := &Client{
		ID:       uuid.New().String(),
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		IsTyping: false,
		LastSeen: time.Now(),
	}

	hub.register <- client

	go client.writePump()
	go client.readPump(hub)
}

func main() {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	http.Handle("/", http.FileServer(http.Dir("./static/")))

	fmt.Println("🚀 Chat server starting on :9090")
	fmt.Println("📱 Visit http://localhost:9090 to start chatting!")

	log.Fatal(http.ListenAndServe(":9090", nil))
}
