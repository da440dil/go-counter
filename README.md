# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/da440dil/go-counter.svg)](https://pkg.go.dev/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting using [Redis](https://redis.io/).

[Example](./examples/limiter/main.go) usage:

```go 
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
```
