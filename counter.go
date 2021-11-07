// Package counter provides functions for distributed rate limiting.
package counter

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient is redis scripter interface.
type RedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
}

// Result of Count() operation.
type Result struct {
	counter int64
	ttl     int64
}

// OK is operation success flag.
func (r Result) OK() bool {
	return r.ttl == -1
}

// Counter after increment.
// With fixed window algorithm in use counter is current window counter.
// With sliding window algorithm in use counter is sliding window counter.
func (r Result) Counter() int {
	return int(r.counter)
}

// TTL of the current window.
// Makes sense if operation failed, otherwise ttl is less than 0.
func (r Result) TTL() time.Duration {
	return time.Duration(r.ttl) * time.Millisecond
}

// ErrUnexpectedRedisResponse is the error returned when Redis command returns response of unexpected type.
var ErrUnexpectedRedisResponse = errors.New("counter: unexpected redis response")

// Counter implements distributed rate limiting.
type Counter struct {
	client RedisClient
	script *redis.Script
	size   int
	limit  int
}

func newCounter(client RedisClient, size time.Duration, limit uint, src string) *Counter {
	return &Counter{client, redis.NewScript(src), int(size / time.Millisecond), int(limit)}
}

// Count increments key by value.
func (c *Counter) Count(ctx context.Context, key string, value int) (Result, error) {
	r := Result{}
	res, err := c.script.Run(ctx, c.client, []string{key}, value, c.size, c.limit).Result()
	if err != nil {
		return r, err
	}
	arr, ok := res.([]interface{})
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	if len(arr) != 2 {
		return r, ErrUnexpectedRedisResponse
	}
	r.counter, ok = arr[0].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	r.ttl, ok = arr[1].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	return r, nil
}

//go:embed fixedwindow.lua
var fwsrc string

// FixedWindow creates new counter which implements distributed rate limiting using fixed window algorithm.
func FixedWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return newCounter(client, size, limit, fwsrc)
}

//go:embed slidingwindow.lua
var swsrc string

// SlidingWindow creates new counter which implements distributed rate limiting using sliding window algorithm.
func SlidingWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return newCounter(client, size, limit, swsrc)
}
