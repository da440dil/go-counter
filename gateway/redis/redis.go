// Package redis implements Gateway to Redis to store a counter value.
package redis

import (
	"github.com/go-redis/redis"
)

type gatewayError string

func (e gatewayError) Error() string {
	return string(e)
}

// ErrInvalidResponse is the error returned when Redis command returns response of invalid type.
const ErrInvalidResponse = gatewayError("counter/gateway/redis: invalid response")

// ErrKeyNameClash is the error returned when Redis key exists and has no TTL.
const ErrKeyNameClash = gatewayError("counter/gateway/redis: key name clash")

var incr = redis.NewScript(
	"local v = redis.call(\"incr\", KEYS[1]) " +
		"if v == 1 then " +
		"redis.call(\"pexpire\", KEYS[1], ARGV[1]) " +
		"return {v, -2} " +
		"end " +
		"local t = redis.call(\"pttl\", KEYS[1]) " +
		"return {v, t}",
)

// Gateway to Redis storage.
type Gateway struct {
	client *redis.Client
}

// New creates new Gateway.
func New(client *redis.Client) *Gateway {
	return &Gateway{client}
}

// Incr sets key value and TTL of key if key not exists.
// Increments key value if key exists.
// Returns key value after increment.
// Returns TTL of a key in milliseconds.
func (gw *Gateway) Incr(key string, ttl int) (int, int, error) {
	res, err := incr.Run(gw.client, []string{key}, ttl).Result()
	if err != nil {
		return 0, 0, err
	}

	var ok bool
	var arr []interface{}
	arr, ok = res.([]interface{})
	if !ok {
		return 0, 0, ErrInvalidResponse
	}
	if len(arr) != 2 {
		return 0, 0, ErrInvalidResponse
	}

	var v int64
	v, ok = arr[0].(int64)
	if !ok {
		return 0, 0, ErrInvalidResponse
	}

	var t int64
	t, ok = arr[1].(int64)
	if !ok {
		return 0, 0, ErrInvalidResponse
	}

	if t == -1 {
		return 0, 0, ErrKeyNameClash
	}

	if t == -2 {
		return int(v), ttl, nil
	}

	return int(v), int(t), nil
}
