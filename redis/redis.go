// Package redis implements Gateway to Redis to store a counter value.
package redis

import (
	"errors"

	"github.com/go-redis/redis"
)

// ErrInvalidResponse is the error returned when Redis command returns response of invalid type.
var ErrInvalidResponse = errors.New("Invalid response")

// ErrKeyNameClash is the error returned when Redis key exists and has no TTL.
var ErrKeyNameClash = errors.New("Key name clash")

var incr = redis.NewScript(
	"local v = redis.call(\"incr\", KEYS[1]) " +
		"if v > tonumber(ARGV[1]) then " +
		"return redis.call(\"pttl\", KEYS[1]) " +
		"end " +
		"if v == 1 then " +
		"redis.call(\"pexpire\", KEYS[1], ARGV[2]) " +
		"end " +
		"return nil",
)

// Gateway is a gateway to Redis storage.
type Gateway struct {
	client *redis.Client
}

// NewGateway creates new Gateway.
func NewGateway(client *redis.Client) *Gateway {
	return &Gateway{client}
}

func (gw *Gateway) Incr(key string, limit int64, ttl int64) (int64, error) {
	res, err := incr.Run(gw.client, []string{key}, limit, ttl).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, nil
		}
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
