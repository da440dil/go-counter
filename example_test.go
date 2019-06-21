package counter_test

import (
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/go-redis/redis"
)

func Example() {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctr := counter.NewCounter(
		client,
		counter.Params{TTL: time.Millisecond * 100, Limit: 1},
	)
	handle := func(err error) {
		if err == nil {
			fmt.Println("Counter has counted the key")
		} else {
			if e, ok := err.(counter.TTLError); ok {
				fmt.Printf("Counter has reached the limit, retry after %v\n", e.TTL())
			} else {
				panic(err)
			}
		}
	}
	key := "key"
	handle(ctr.Count(key))
	handle(ctr.Count(key))
	// Output:
	// Counter has counted the key
	// Counter has reached the limit, retry after 100ms
}
