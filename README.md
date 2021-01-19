# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/da440dil/go-counter.svg)](https://pkg.go.dev/github.com/da440dil/go-counter)
[![GoDoc](https://godoc.org/github.com/da440dil/go-counter?status.svg)](https://godoc.org/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting using [Redis](https://redis.io/).

[Example](./examples/fixedwindow/main.go) using [fixed window](./fixedwindow.go) algorithm:

```go 
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
		fmt.Printf("Value: %v, result: { ok: %v, counter: %v, ttl: %v }\n", v, r.OK(), r.Counter(), r.TTL())
	}
	count(101)
	count(20)
	count(30)
	count(51)
	time.Sleep(time.Second) // wait for the next window to start
	count(70)
	// Output:
	// Value: 101, result: { ok: false, counter: 0, ttl: -2ms }
	// Value: 20, result: { ok: true, counter: 20, ttl: -1ms }
	// Value: 30, result: { ok: true, counter: 50, ttl: -1ms }
	// Value: 51, result: { ok: false, counter: 50, ttl: 999ms }
	// Value: 70, result: { ok: true, counter: 70, ttl: -1ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
```

[Example](./examples/slidingwindow/main.go) using [sliding window](./slidingwindow.go) algorithm:

```go
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
		fmt.Printf("Value: %v, result: { ok: %v, counter: %v, ttl: %v }\n", v, r.OK(), r.Counter(), r.TTL())
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
	// Value: 101, result: { ok: false, counter: 0, ttl: 734ms }
	// Value: 20, result: { ok: true, counter: 20, ttl: -1ms }
	// Value: 30, result: { ok: true, counter: 50, ttl: -1ms }
	// Value: 51, result: { ok: false, counter: 50, ttl: 997ms }
	// Value: 70, result: { ok: false, counter: 49, ttl: 996ms }
	// Value: 70, result: { ok: true, counter: 84, ttl: -1ms }
}

func requireNoError(err error) {
	if err != nil {
		panic(err)
	}
}
```
