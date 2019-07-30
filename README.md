# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-counter?status.svg)](https://godoc.org/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting using [Redis](https://redis.io/).

## Example

```go
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/go-redis/redis"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctr := counter.NewCounter(
		client,
		counter.Params{TTL: time.Millisecond * 100, Limit: 2},
	)
	key := "key"
	var wg sync.WaitGroup
	count := func() {
		wg.Add(1)
		go func() {
			v, err := ctr.Count(key)
			if err == nil {
				fmt.Printf("Counter has counted the key, remainder %v\n", v)
			} else {
				if e, ok := err.(counter.TTLError); ok {
					fmt.Printf("Counter has reached the limit, retry after %v\n", e.TTL())
				} else {
					panic(err)
				}
			}
			wg.Done()
		}()
	}

	count() // Counter has counted the key, remainder 1
	count() // Counter has counted the key, remainder 0
	count() // Counter has reached the limit, retry after 100ms
	wg.Wait()
}
```