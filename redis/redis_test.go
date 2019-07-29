package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const Addr = "localhost:6379"
const DB = 10

const Key = "key"
const TTL = 100

func TestGateway(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	storage := &Storage{client, t}
	storage.Del(Key)
	defer storage.Del(Key)

	timeout := time.Duration(TTL+20) * time.Millisecond

	t.Run("set key value and TTL of key if key not exists", func(t *testing.T) {
		gw := NewGateway(client)

		v, ttl, err := gw.Incr(Key, TTL)
		assert.NoError(t, err)
		assert.Equal(t, 1, v)
		assert.Equal(t, TTL, ttl)

		k := storage.Get(Key)
		assert.Equal(t, "1", k)
		r := storage.PTTL(Key)
		assert.Greater(t, r, 0)
		assert.LessOrEqual(t, r, TTL)

		time.Sleep(timeout)

		k = storage.Get(Key)
		assert.Equal(t, "", k)
		r = storage.PTTL(Key)
		assert.Equal(t, -2, r)
	})

	t.Run("increment key value if key exists", func(t *testing.T) {
		gw := NewGateway(client)
		gw.Incr(Key, TTL)

		v, ttl, err := gw.Incr(Key, TTL)
		assert.NoError(t, err)
		assert.Equal(t, 2, v)
		assert.Greater(t, ttl, 0)
		assert.LessOrEqual(t, ttl, TTL)

		k := storage.Get(Key)
		assert.Equal(t, "2", k)
		r := storage.PTTL(Key)
		assert.Greater(t, r, 0)
		assert.LessOrEqual(t, r, TTL)

		time.Sleep(timeout)

		k = storage.Get(Key)
		assert.Equal(t, "", k)
		r = storage.PTTL(Key)
		assert.Equal(t, -2, r)
	})
}

func BenchmarkGateway(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	testCases := []struct {
		ttl int
	}{
		{1000},
		{10000},
		{100000},
		{1000000},
	}

	storage := &Storage{client, b}
	gw := NewGateway(client)

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("ttl %v", tc.ttl), func(b *testing.B) {
			storage.Del(keys...)
			defer storage.Del(keys...)

			ttl := tc.ttl
			kl := len(keys)
			for i := 0; i < b.N; i++ {
				_, _, err := gw.Incr(keys[i%kl], ttl)
				assert.NoError(b, err)
			}
		})
	}
}

type Storage struct {
	c *redis.Client
	t interface{ Fatal(...interface{}) }
}

func (s *Storage) Del(keys ...string) {
	if err := s.c.Del(keys...).Err(); err != nil {
		s.t.Fatal("redis del failed")
	}
}

func (s *Storage) Get(key string) string {
	v, err := s.c.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return ""
		}
		s.t.Fatal("redis get failed")
	}
	return v
}

func (s *Storage) PTTL(key string) int {
	v, err := s.c.PTTL(key).Result()
	if err != nil {
		s.t.Fatal("redis pttl failed")
	}
	return int(v / time.Millisecond)
}
