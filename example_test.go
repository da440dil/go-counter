package counter_test

import (
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
)

func ExampleCounter() {
	c, err := counter.New(2, time.Millisecond*100)
	if err != nil {
		panic(err)
	}
	key := "key"
	count := func(n int) {
		v, err := c.Count(key)
		if err == nil {
			fmt.Printf("Counter #%v has counted the key, remainder %v\n", n, v)
		} else {
			if _, ok := err.(counter.TTLError); ok {
				fmt.Printf("Counter #%v has reached the limit", n)
			} else {
				panic(err)
			}
		}
	}

	count(1)
	count(2)
	count(3)
	// Output:
	// Counter #1 has counted the key, remainder 1
	// Counter #2 has counted the key, remainder 0
	// Counter #3 has reached the limit
}
