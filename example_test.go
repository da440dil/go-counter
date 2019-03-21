package counter_test

import (
	"fmt"
	"time"

	"github.com/da440dil/go-counter"
	storage "github.com/da440dil/go-counter/redis"
	"github.com/go-redis/redis"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func Example() {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	const (
		limit = 2
		ttl   = time.Millisecond * 100
		key   = "key"
	)
	// Create Redis storage
	storage := storage.NewStorage(client)
	params := counter.Params{
		Limit: limit,
		TTL:   ttl,
	}
	// Create first counter
	counter1 := Counter{
		counter: counter.NewCounter(storage, params),
		key:     key,
		id:      1,
	}
	// Create second counter
	counter2 := Counter{
		counter: counter.NewCounter(storage, params),
		key:     key,
		id:      2,
	}

	counter1.Count() // Counter#1 has counted the key
	counter2.Count() // Counter#2 has counted the key
	counter1.Count() // Counter#1 has reached the limit, retry after 99 ms
	counter2.Count() // Counter#2 has reached the limit, retry after 98 ms
	time.Sleep(time.Millisecond * 200)
	fmt.Println("Timeout 200 ms is up")
	counter1.Count() // Counter#1 has counted the key
	counter2.Count() // Counter#2 has counted the key
}

type Counter struct {
	counter *counter.Counter
	key     string
	id      int
}

func (c Counter) Count() {
	v, err := c.counter.Count(c.key)
	if err != nil {
		panic(err)
	}
	if v == -1 {
		fmt.Printf("Counter#%d has counted the key\n", c.id)
	} else {
		fmt.Printf("Counter#%d has reached the limit, retry after %d ms\n", c.id, v)
	}
}
