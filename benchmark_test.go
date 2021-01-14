package counter

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func BenchmarkCounter(b *testing.B) {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	size := 10 * time.Second
	limit := 10000
	tests := map[string]*Counter{
		"FixedWindow":   FixedWindow(client, size, limit),
		"SlidingWindow": SlidingWindow(client, size, limit),
	}

	ctx := context.Background()
	key := "key"
	value := 1
	for name, tc := range tests {
		b.Run(name, func(b *testing.B) {
			err := client.Del(ctx, key).Err()
			if err != nil {
				b.Fatal(err)
			}
			for i := 0; i < b.N; i++ {
				tc.Count(ctx, key, value)
			}
		})
	}
}
