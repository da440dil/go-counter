package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-counter"
	gw "github.com/da440dil/go-counter/gateway/redis"
	"github.com/go-redis/redis"
)

func main() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	g := gw.New(client)
	c, err := counter.New(2, time.Millisecond*100, counter.WithGateway(g))
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
