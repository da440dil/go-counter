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

// Result is counter value increment result.
type Result struct {
	counter int64
	ttl     int64
	limit   int64
}

// OK is operation success flag.
func (r Result) OK() bool {
	return r.ttl == -1
}

// Counter is current counter value.
func (r Result) Counter() int64 {
	return r.counter
}

// Remainder is diff between limit and current counter value.
func (r Result) Remainder() int64 {
	return r.limit - r.counter
}

// TTL of the current window.
// Makes sense if operation failed, otherwise ttl is less than 0.
func (r Result) TTL() time.Duration {
	return time.Duration(r.ttl) * time.Millisecond
}

// ErrUnexpectedRedisResponse is the error returned when Redis command returns response of unexpected type.
var ErrUnexpectedRedisResponse = errors.New("counter: unexpected redis response")

// Counter implements distributed counter.
type Counter struct {
	client RedisClient
	script *redis.Script
	limit  int64
	size   int
}

// Count increments key value by specified value.
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
	r.limit = c.limit
	return r, nil
}

//go:embed fixedwindow.lua
var fwsrc string
var fwscr = redis.NewScript(fwsrc)

// FixedWindow creates new counter which implements distributed counter using fixed window algorithm.
func FixedWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return &Counter{client: client, script: fwscr, size: int(size / time.Millisecond), limit: int64(limit)}
}

//go:embed slidingwindow.lua
var swsrc string
var swscr = redis.NewScript(swsrc)

// SlidingWindow creates new counter which implements distributed counter using sliding window algorithm.
func SlidingWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return &Counter{client: client, script: swscr, size: int(size / time.Millisecond), limit: int64(limit)}
}
