package counter_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/go-redis/redis"
)

func Example() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	c, err := counter.NewCounter(client, 2, time.Millisecond*100)
	if err != nil {
		panic(err)
	}
	key := "key"
	var wg sync.WaitGroup
	count := func() {
		wg.Add(1)
		go func() {
			v, err := c.Count(key)
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
