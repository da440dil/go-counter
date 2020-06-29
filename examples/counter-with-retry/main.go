package main

import (
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/da440dil/go-trier"
)

func main() {
	// Create counter
	c, err := counter.New(2, time.Millisecond*20)
	if err != nil {
		panic(err)
	}
	// Create trier
	t, err := trier.New(
		// Set maximum number of retries
		trier.WithRetryCount(1),
		// Set delay between retries
		trier.WithRetryDelay(time.Millisecond*40),
	)
	if err != nil {
		panic(err)
	}
	// Create retriable function
	fn := func(n int) func() (bool, time.Duration, error) {
		return func() (bool, time.Duration, error) {
			v, err := c.Count("key")
			if err == nil {
				fmt.Printf("Counter #%v has counted the key, remainder %v\n", n, v)
				return true, -1, nil // Success
			}
			if e, ok := err.(*counter.TTLError); ok {
				fmt.Printf("Counter #%v has reached the limit, retry after %v\n", n, e.TTL())
				return false, e.TTL(), nil // Failure
			}
			return false, -1, err // Error
		}
	}
	for i := 1; i < 4; i++ {
		// Execute function
		if err = t.Try(fn(i)); err != nil {
			if e, ok := err.(*trier.TTLError); ok {
				fmt.Printf("Number of retries with counter #%v exceeded, retry after %v\n", i, e.TTL())
			} else {
				panic(err)
			}
		}
	}
	// Output:
	// Counter #1 has counted the key, remainder 1
	// Counter #2 has counted the key, remainder 0
	// Counter #3 has reached the limit, retry after 20ms
	// Counter #3 has counted the key, remainder 1
}
