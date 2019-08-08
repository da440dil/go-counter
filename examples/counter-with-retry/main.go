package main

import (
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
	"github.com/da440dil/go-runner"
)

func main() {
	// Create counter
	c, err := counter.New(2, time.Millisecond*20)
	if err != nil {
		panic(err)
	}
	// Create runner
	r, err := runner.New(
		// Set maximum number of retries
		runner.WithRetryCount(1),
		// Set delay between retries
		runner.WithRetryDelay(time.Millisecond*40),
	)
	if err != nil {
		panic(err)
	}
	// Create retriable function
	fn := func(n int) func() (bool, error) {
		return func() (bool, error) {
			v, err := c.Count("key")
			if err == nil {
				fmt.Printf("Counter #%v has counted the key, remainder %v\n", n, v)
				return true, nil // Success
			}
			if e, ok := err.(counter.TTLError); ok {
				fmt.Printf("Counter #%v has reached the limit, retry after %v\n", n, e.TTL())
				return false, nil // Failure
			}
			return false, err // Error
		}
	}
	for i := 1; i < 4; i++ {
		// Run function
		if err = r.Run(fn(i)); err != nil {
			panic(err)
		}
	}
	// Output:
	// Counter #1 has counted the key, remainder 1
	// Counter #2 has counted the key, remainder 0
	// Counter #3 has reached the limit, retry after 20ms
	// Counter #3 has counted the key, remainder 1
}
