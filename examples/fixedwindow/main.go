package main

import (
	"context"
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/go-redis/redis/v8"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	requireNoError(err)

	c := counter.FixedWindow(client, time.Second, 100)

	count := func(v int) {
		r, err := c.Count(ctx, key, v)
		requireNoError(err)
		fmt.Printf(
			"Value: %v, result: { ok: %v, counter: %v, remainder: %v, ttl: %v }\n",
			v, r.OK(), r.Counter(), r.Remainder(), r.TTL(),
		)
	}
	count(101)
	count(20)
	count(30)
	count(51)
	time.Sleep(time.Second) // wait for the next window to start
	count(70)
	// Output:
	// Value: 101, result: { ok: false, counter: 0, remainder: 100, ttl: 0s }
	// Value: 20, result: { ok: true, counter: 20, remainder: 80, ttl: -1ms }
	// Value: 30, result: { ok: true, counter: 50, remainder: 50, ttl: -1ms }
	// Value: 51, result: { ok: false, counter: 50, remainder: 50, ttl: 999ms }
	// Value: 70, result: { ok: true, counter: 70, remainder: 30, ttl: -1ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
