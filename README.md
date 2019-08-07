# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-counter?status.svg)](https://godoc.org/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting with pluggable storage to store a counters state.

## Example

```go
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-counter"
)

func main() {
	c, err := counter.New(2, time.Millisecond*100)
	if err != nil {
		panic(err)
	}
	key := "key"
	var wg sync.WaitGroup
	count := func(n int) {
		wg.Add(1)
		go func() {
			v, err := c.Count(key)
			if err == nil {
				fmt.Printf("Counter #%v has counted the key, remainder %v\n", n, v)
			} else {
				if e, ok := err.(counter.TTLError); ok {
					fmt.Printf("Counter #%v has reached the limit, retry after %v\n", n, e.TTL())
				} else {
					panic(err)
				}
			}
			wg.Done()
		}()
	}

	count(1) // Counter #1 has counted the key, remainder 1
	count(2) // Counter #2 has counted the key, remainder 0
	count(3) // Counter #3 has reached the limit, retry after 100ms
	wg.Wait()
}
```