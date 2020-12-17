// Package counter provides functions for distributed rate limiting.
package counter

import (
	"context"
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

var errInvalidResponse = errors.New("counter: invalid redis response")

// Result of count() operation.
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

// Counter implements distributed rate limiting.
type Counter struct {
	client RedisClient
	size   int
	limit  int
	script *redis.Script
}

func newCounter(client RedisClient, size time.Duration, limit int, src string) *Counter {
	return &Counter{client, int(size / time.Millisecond), limit, redis.NewScript(src)}
}

// Count increments key by value.
func (c *Counter) Count(ctx context.Context, key string, value int) (Result, error) {
	r := Result{}
	res, err := c.script.Run(ctx, c.client, []string{key}, value, c.size, c.limit).Result()
	if err != nil {
		return r, err
	}
	var arr []interface{}
	var ok bool
	arr, ok = res.([]interface{})
	if !ok {
		return r, errInvalidResponse
	}
	if len(arr) != 2 {
		return r, errInvalidResponse
	}
	r.counter, ok = arr[0].(int64)
	if !ok {
		return r, errInvalidResponse
	}
	r.ttl, ok = arr[1].(int64)
	if !ok {
		return r, errInvalidResponse
	}
	return r, nil
}
