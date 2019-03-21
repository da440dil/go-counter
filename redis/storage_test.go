package redis

import (
	"testing"
	"time"

	rd "github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const redisAddr = "127.0.0.1:6379"
const redisDb = 10

func Test(t *testing.T) {
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
		limit = 2
		ttl   = time.Millisecond * 1000
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(client)

	var err error
	var v int64

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}

func TestTTL(t *testing.T) {
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
		limit = 2
		ttl   = time.Millisecond * 100
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(client)

	var err error
	var v int64

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	time.Sleep(time.Millisecond * 200)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	if err := client.Del(key).Err(); err != nil {
		t.Fatal("redis del failed")
	}
}
