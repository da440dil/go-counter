package main

import (
	"context"
	"fmt"
	"sync"
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

	// Create limiter suite with 2 limiters.
	ls := counter.NewLimiterSuite(
		// First limiter is limited to 3 calls per second.
		counter.NewLimiter(counter.FixedWindow(client, time.Second, 3)),
		// Second limiter is limited to 5 calls per 2 seconds.
		counter.NewLimiter(counter.FixedWindow(client, time.Second*2, 5)),
	)

	limit := func() {
		r, err := ls.Limit(ctx, key)
		requireNoError(err)
		fmt.Printf(
			"Result: { ok: %v, counter: %v, remainder: %v, ttl: %v }\n",
			r.OK(), r.Counter(), r.Remainder(), r.TTL(),
		)
	}
	limitN := func(n int) {
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				limit()
			}()
		}
		wg.Wait()
	}

	limitN(4)
	time.Sleep(time.Second) // wait for the next window to start
	limitN(2)
	// Output:
	// Result: { ok: true, counter: 1, remainder: 2, ttl: -1ms }
	// Result: { ok: true, counter: 2, remainder: 1, ttl: -1ms }
	// Result: { ok: true, counter: 3, remainder: 0, ttl: -1ms }
	// Result: { ok: false, counter: 3, remainder: 0, ttl: 999ms }
	// Result: { ok: true, counter: 5, remainder: 0, ttl: -1ms }
	// Result: { ok: false, counter: 5, remainder: 0, ttl: 989ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
