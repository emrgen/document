package cache

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client redis.Client
}

func NewRedis() (Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})

	return &Redis{client: client}, nil
}

func (r *Redis) Set(ctx context.Context, k string, v any, ttl time.Duration) error {
	// marshal
	value, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// r.client.Set()

	return nil
}
