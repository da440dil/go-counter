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

	// Create limiter with 2 limits.
	limiter := counter.NewLimiter(
		client,
		// First limit: no more than 3 limiter calls within 1 second.
		counter.WithLimit(time.Second, 3),
		// Second limit: no more than 5 limiter calls within 2 seconds.
		counter.WithLimit(time.Second*2, 5),
	)

	limit := func() {
		r, err := limiter.Limit(ctx, key)
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
	// Result: { ok: true, counter: 1, remainder: 2, ttl: 1s }
	// Result: { ok: true, counter: 3, remainder: 0, ttl: 998ms }
	// Result: { ok: true, counter: 2, remainder: 1, ttl: 998ms }
	// Result: { ok: false, counter: 3, remainder: 0, ttl: 998ms }
	// Result: { ok: true, counter: 5, remainder: 0, ttl: 993ms }
	// Result: { ok: false, counter: 5, remainder: 0, ttl: 993ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
```
