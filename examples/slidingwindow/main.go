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

	c := counter.SlidingWindow(client, time.Second, 100)

	count := func(v int) counter.Result {
		r, err := c.Count(ctx, key, v)
		requireNoError(err)
		fmt.Printf(
			"Value: %v, result: { ok: %v, counter: %v, remainder: %v, ttl: %v }\n",
			v, r.OK(), r.Counter(), r.Remainder(), r.TTL(),
		)
		return r
	}
	r := count(101)
	time.Sleep(r.TTL()) // wait for the next window to start
	count(20)
	count(30)
	count(51)
	time.Sleep(time.Second) // wait for the next window to start
	count(70)
	time.Sleep(700 * time.Millisecond) // wait for the most time of the current window to pass
	count(70)
	// Output:
	// Value: 101, result: { ok: false, counter: 0, remainder: 100, ttl: 101ms }
	// Value: 20, result: { ok: true, counter: 20, remainder: 80, ttl: 997ms }
	// Value: 30, result: { ok: true, counter: 50, remainder: 50, ttl: 995ms }
	// Value: 51, result: { ok: false, counter: 50, remainder: 50, ttl: 993ms }
	// Value: 70, result: { ok: false, counter: 49, remainder: 51, ttl: 987ms }
	// Value: 70, result: { ok: true, counter: 83, remainder: 17, ttl: 264ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
