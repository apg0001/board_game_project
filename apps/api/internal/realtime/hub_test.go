package realtime

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeBus struct {
	published chan Message
}

func (b *fakeBus) Publish(_ context.Context, message Message) error {
	b.published <- message
	return nil
}

func (b *fakeBus) Subscribe(_ context.Context, _ chan<- Message) {}

func (b *fakeBus) Close() error {
	return nil
}

func TestHubPublishesLocalBroadcastToBus(t *testing.T) {
	bus := &fakeBus{published: make(chan Message, 1)}
	hub := NewHub(slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil)), WithBus(bus, "node-a"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	hub.Broadcast(Message{Room: "room:r1", Type: "room.updated"})

	select {
	case message := <-bus.published:
		if message.Origin != "node-a" {
			t.Fatalf("expected node origin, got %s", message.Origin)
		}
	case <-time.After(time.Second):
		t.Fatal("expected bus publish")
	}
}
