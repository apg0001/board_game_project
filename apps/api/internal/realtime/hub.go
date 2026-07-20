package realtime

import (
	"context"
	"log/slog"
)

type Message struct {
	Room    string `json:"room"`
	Type    string `json:"type"`
	Payload any    `json:"payload"`
	Origin  string `json:"origin,omitempty"`
}

type Client struct {
	id   string
	room string
	send chan Message
}

type Hub struct {
	logger     *slog.Logger
	nodeID     string
	bus        Bus
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	remote     chan Message
	clients    map[*Client]struct{}
	rooms      map[string]map[*Client]struct{}
}

type Bus interface {
	Publish(ctx context.Context, message Message) error
	Subscribe(ctx context.Context, messages chan<- Message)
	Close() error
}

func NewHub(logger *slog.Logger, options ...Option) *Hub {
	hub := &Hub{
		logger:     logger,
		nodeID:     "local",
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 64),
		remote:     make(chan Message, 64),
		clients:    make(map[*Client]struct{}),
		rooms:      make(map[string]map[*Client]struct{}),
	}
	for _, option := range options {
		option(hub)
	}
	return hub
}

type Option func(*Hub)

func WithBus(bus Bus, nodeID string) Option {
	return func(h *Hub) {
		h.bus = bus
		if nodeID != "" {
			h.nodeID = nodeID
		}
	}
}

func (h *Hub) Broadcast(message Message) {
	if message.Origin == "" {
		message.Origin = h.nodeID
	}
	h.broadcast <- message
}

func (h *Hub) Run(ctx context.Context) {
	if h.bus != nil {
		go h.bus.Subscribe(ctx, h.remote)
	}
	defer func() {
		if h.bus != nil {
			_ = h.bus.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.clients[client] = struct{}{}
			if h.rooms[client.room] == nil {
				h.rooms[client.room] = make(map[*Client]struct{})
			}
			h.rooms[client.room][client] = struct{}{}
			h.logger.Info("websocket client joined", "room", client.room)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			if message.Origin == "" {
				message.Origin = h.nodeID
			}
			h.deliver(message)
			if h.bus != nil {
				if err := h.bus.Publish(ctx, message); err != nil {
					h.logger.Warn("redis realtime publish failed", "error", err)
				}
			}
		case message := <-h.remote:
			if message.Origin == h.nodeID {
				continue
			}
			h.deliver(message)
		}
	}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		delete(h.rooms[client.room], client)
		close(client.send)
	}
}

func (h *Hub) deliver(message Message) {
	for client := range h.rooms[message.Room] {
		select {
		case client.send <- message:
		default:
			h.unregisterClient(client)
		}
	}
}
