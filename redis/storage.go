// Package redis is for creating storage using Redis.
package redis

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

// ErrInvalidResponse is the error returned when Redis command returns response of invalid type.
var ErrInvalidResponse = errors.New("Invalid response")

// ErrKeyNameClash is the error returned when Redis key exists and has no TTL.
var ErrKeyNameClash = errors.New("Key name clash")

var incr = redis.NewScript(`local v = redis.call("incr", KEYS[1]) if v > tonumber(ARGV[1]) then return redis.call("pttl", KEYS[1]) end if v == 1 then redis.call("pexpire", KEYS[1], ARGV[2]) end return nil`)

// NewStorage allocates and returns new Storage.
func NewStorage(client *redis.Client) *Storage {
	return &Storage{
		client: client,
	}
}

// Storage implements storage using Redis.
type Storage struct {
	client *redis.Client
}

func (s *Storage) Incr(key string, limit uint64, ttl time.Duration) (int64, error) {
	res, err := incr.Run(s.client, []string{key}, limit, strconv.FormatInt(int64(ttl/time.Millisecond), 10)).Result()
	if err == redis.Nil {
		return -1, nil
	}
	if err != nil {
		return -2, err
	}
	i, ok := res.(int64)
	if !ok {
		return -2, ErrInvalidResponse
	}
	if i == -1 {
		return -2, ErrKeyNameClash
	}
	return i, nil
}
