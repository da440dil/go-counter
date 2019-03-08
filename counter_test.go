package counter

import (
	"testing"
	"time"

	rs "github.com/da440dil/counter/redis"
	rd "github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func TestRedis(t *testing.T) {
	client := rd.NewClient(&rd.Options{
		Addr: redisAddr,
		DB:   redisDb,
	})
	defer client.Close()

	if err := client.Ping().Err(); err != nil {
		t.Fatal("redis ping failed")
	}

	const key = "key"

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}

	const (
		limit uint64 = 2
		ttl          = time.Millisecond * 200
	)

	st := rs.NewStorage(client)
	ct := NewCounter(st, Params{
		Limit: limit,
		TTL:   ttl,
	})

	var err error
	var v int64

	v, err = ct.Count(key)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = ct.Count(key)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = ct.Count(key)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= int64(ttl))

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}

	v, err = ct.Count(key)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}
