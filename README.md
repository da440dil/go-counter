# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-counter?status.svg)](https://godoc.org/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting with pluggable storage to store a counters state.

## Basic usage

```go
// Create new Counter
c, _ := counter.New(1, time.Millisecond*100)
// Increment counter and get remainder
if v, err := c.Count("key"); err != nil {
	if e, ok := err.(locker.TTLError); ok {
		// Use e.TTL() if need
	} else {
		// Handle err
	}
} else {
	// Counter value equals 1
	// Remainder (v) equals 0
	// Next c.Count("key") call will return TTLError
}
```

## Example usage

- [example](./examples/counter-gateway-default/main.go) usage with default [gateway](./gateway/memory/memory.go)
- [example](./examples/counter-gateway-memory/main.go) usage with memory [gateway](./gateway/memory/memory.go)
- [example](./examples/counter-gateway-redis/main.go) usage with [Redis](https://redis.io) [gateway](./gateway/redis/redis.go)
- [example](./examples/counter-with-retry/main.go) usage with [retry](https://github.com/da440dil/go-trier)