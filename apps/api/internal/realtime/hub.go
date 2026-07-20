package realtime

import "log/slog"

type Message struct {
	Room    string `json:"room"`
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Client struct {
	id   string
	room string
	send chan Message
}

type Hub struct {
	logger     *slog.Logger
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	clients    map[*Client]struct{}
	rooms      map[string]map[*Client]struct{}
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		logger:     logger,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 64),
		clients:    make(map[*Client]struct{}),
		rooms:      make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Broadcast(message Message) {
	h.broadcast <- message
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
			if h.rooms[client.room] == nil {
				h.rooms[client.room] = make(map[*Client]struct{})
			}
			h.rooms[client.room][client] = struct{}{}
			h.logger.Info("websocket client joined", "room", client.room)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.rooms[client.room], client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.rooms[message.Room] {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					delete(h.rooms[client.room], client)
					close(client.send)
				}
			}
		}
	}
}
