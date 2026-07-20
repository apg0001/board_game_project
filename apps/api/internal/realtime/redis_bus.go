package realtime

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

const redisChannel = "boardgame:realtime"

type RedisBus struct {
	client *redis.Client
}

func NewRedisBus(redisURL string) (*RedisBus, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisBus{client: redis.NewClient(options)}, nil
}

func (b *RedisBus) Ping(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}

func (b *RedisBus) Publish(ctx context.Context, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, redisChannel, body).Err()
}

func (b *RedisBus) Subscribe(ctx context.Context, messages chan<- Message) {
	pubsub := b.client.Subscribe(ctx, redisChannel)
	defer pubsub.Close()

	channel := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case received, ok := <-channel:
			if !ok {
				return
			}
			var message Message
			if err := json.Unmarshal([]byte(received.Payload), &message); err == nil {
				messages <- message
			}
		}
	}
}

func (b *RedisBus) Close() error {
	return b.client.Close()
}
