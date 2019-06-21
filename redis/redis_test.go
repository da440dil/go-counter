package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

const Addr = "127.0.0.1:6379"
const DB = 10

func TestGateway(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	const (
		key     = "key"
		limit   = int64(2)
		ttlTime = time.Millisecond * 500
		ttl     = int64(ttlTime / time.Millisecond)
		vOK     = int64(-1)
		zeroTTL = int64(0)
		nilTTL  = int64(-2)
	)

	storage := &Storage{client, t}
	storage.Del(key)
	defer storage.Del(key)

	gw := NewGateway(client)

	t.Run("incr #1 success", func(t *testing.T) {
		v, err := gw.Incr(key, limit, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, "1", k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("incr #2 success", func(t *testing.T) {
		v, err := gw.Incr(key, limit, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, "2", k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("incr #3 fail", func(t *testing.T) {
		v, err := gw.Incr(key, limit, ttl)
		assert.NoError(t, err)
		assert.Greater(t, v, zeroTTL)
		assert.LessOrEqual(t, v, ttl)
		k := storage.Get(key)
		assert.Equal(t, "3", k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})

	t.Run("sleep", func(t *testing.T) {
		time.Sleep(ttlTime + time.Millisecond*100)
		k := storage.Get(key)
		assert.Equal(t, "", k)
		r := storage.PTTL(key)
		assert.Equal(t, nilTTL, r)
	})

	t.Run("incr #1 success", func(t *testing.T) {
		v, err := gw.Incr(key, limit, ttl)
		assert.NoError(t, err)
		assert.Equal(t, vOK, v)
		k := storage.Get(key)
		assert.Equal(t, "1", k)
		r := storage.PTTL(key)
		assert.Greater(t, r, zeroTTL)
		assert.LessOrEqual(t, r, ttl)
	})
}

func BenchmarkGateway(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	testCases := []struct {
		limit int
		ttl   time.Duration
	}{
		{10000, time.Millisecond * 1000},
		{100000, time.Millisecond * 10000},
		{1000000, time.Millisecond * 100000},
		{10000000, time.Millisecond * 1000000},
	}

	storage := &Storage{client, b}
	gw := NewGateway(client)

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("limit %v ttl %v", tc.limit, tc.ttl), func(b *testing.B) {
			storage.Del(keys...)
			defer storage.Del(keys...)

			limit := int64(tc.limit)
			ttl := int64(tc.ttl / time.Millisecond)
			kl := len(keys)
			for i := 0; i < b.N; i++ {
				_, err := gw.Incr(keys[i%kl], limit, ttl)
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

func (s *Storage) PTTL(key string) int64 {
	v, err := s.c.PTTL(key).Result()
	if err != nil {
		s.t.Fatal("redis pttl failed")
	}
	if v > 0 {
		return int64(v / time.Millisecond)
	}
	return int64(v)
}
