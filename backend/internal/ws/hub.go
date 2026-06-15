package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	fileID string
	userID string
	name   string
	color  string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	fileRooms  map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		clients:    make(map[*Client]bool),
		fileRooms:  make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if _, ok := h.fileRooms[client.fileID]; !ok {
				h.fileRooms[client.fileID] = make(map[*Client]bool)
			}
			h.fileRooms[client.fileID][client] = true
			h.mu.Unlock()

			_ = database.AddOnlineUser(client.fileID, client.userID, client.name)

			joinMsg := models.WSMessage{
				Type:      "user-join",
				Payload:   json.RawMessage(`{"name":"` + client.name + `","color":"` + client.color + `"}`),
				UserID:    client.userID,
				Timestamp: time.Now().UnixMilli(),
			}
			joinBytes, _ := json.Marshal(joinMsg)
			h.broadcastToFile(client.fileID, joinBytes, client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if room, ok := h.fileRooms[client.fileID]; ok {
					delete(room, client)
					if len(room) == 0 {
						delete(h.fileRooms, client.fileID)
					}
				}
			}
			h.mu.Unlock()

			_ = database.RemoveOnlineUser(client.fileID, client.userID)

			leaveMsg := models.WSMessage{
				Type:      "user-leave",
				Payload:   json.RawMessage(`{}`),
				UserID:    client.userID,
				Timestamp: time.Now().UnixMilli(),
			}
			leaveBytes, _ := json.Marshal(leaveMsg)
			h.broadcastToFile(client.fileID, leaveBytes, nil)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) broadcastToFile(fileID string, message []byte, exclude *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.fileRooms[fileID]; ok {
		for client := range room {
			if client != exclude {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(room, client)
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		wsMsg.UserID = c.userID
		wsMsg.Timestamp = time.Now().UnixMilli()

		msgBytes, _ := json.Marshal(wsMsg)
		c.hub.broadcastToFile(c.fileID, msgBytes, nil)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
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

func ServeWebSocket(hub *Hub, c *gin.Context) {
	fileID := c.Param("fileId")
	userID := c.Query("userId")
	userName := c.Query("userName")

	if userID == "" {
		userID = uuid.New().String()
	}
	if userName == "" {
		userName = "User" + userID[:8]
	}

	color := generateColor(userID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		fileID: fileID,
		userID: userID,
		name:   userName,
		color:  color,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func generateColor(id string) string {
	var hash uint32
	for i := 0; i < len(id); i++ {
		hash = uint32(id[i]) + ((hash << 5) - hash)
	}
	hue := hash % 360
	return hslToHex(int(hue), 70, 50)
}

func hslToHex(h, s, l int) string {
	c := (1 - abs(2*l/100-1)) * s / 100
	x := c * (1 - abs((h/60)%2-1))
	m := float64(l)/100 - float64(c)/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = float64(c), float64(x), 0
	case h < 120:
		r, g, b = float64(x), float64(c), 0
	case h < 180:
		r, g, b = 0, float64(c), float64(x)
	case h < 240:
		r, g, b = 0, float64(x), float64(c)
	case h < 300:
		r, g, b = float64(x), 0, float64(c)
	default:
		r, g, b = float64(c), 0, float64(x)
	}

	return fmt.Sprintf("#%02x%02x%02x",
		int((r+m)*255),
		int((g+m)*255),
		int((b+m)*255),
	)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func fmt.Sprintf(format string, a ...interface{}) string {
	var buf []byte
	args := a
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'd':
				if len(args) > 0 {
					if v, ok := args[0].(int); ok {
						buf = appendInt(buf, v)
					}
					args = args[1:]
				}
				i++
			case 's':
				if len(args) > 0 {
					if v, ok := args[0].(string); ok {
						buf = append(buf, v...)
					}
					args = args[1:]
				}
				i++
			case 'x':
				if len(args) > 0 {
					if v, ok := args[0].(int); ok {
						buf = appendHex(buf, v)
					}
					args = args[1:]
				}
				i++
			case '%':
				buf = append(buf, '%')
				i++
			default:
				buf = append(buf, format[i])
			}
		} else {
			buf = append(buf, format[i])
		}
	}
	return string(buf)
}

func appendInt(buf []byte, n int) []byte {
	if n < 0 {
		buf = append(buf, '-')
		n = -n
	}
	if n == 0 {
		return append(buf, '0')
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return append(buf, digits[i:]...)
}

func appendHex(buf []byte, n int) []byte {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return append(buf, '0', '0')
	}
	var digits [8]byte
	i := len(digits)
	for n > 0 && i > 0 {
		i--
		d := n % 16
		if d < 10 {
			digits[i] = byte('0' + d)
		} else {
			digits[i] = byte('a' + d - 10)
		}
		n /= 16
	}
	if (len(digits)-i)%2 != 0 {
		i--
		digits[i] = '0'
	}
	return append(buf, digits[i:]...)
}
