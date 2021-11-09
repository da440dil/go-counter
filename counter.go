// Package counter provides functions for distributed rate limiting.
package counter

import (
	"context"
	_ "embed"
	"errors"
	"math/rand"
	"strconv"
	"sync"
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
	limit   int
}

// OK is operation success flag.
func (r Result) OK() bool {
	return r.ttl == -1
}

// Counter is current counter value.
func (r Result) Counter() int {
	return int(r.counter)
}

// Remainder is diff between limit and current counter value.
func (r Result) Remainder() int {
	return r.limit - int(r.counter)
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
	r.limit = c.limit
	if r.ttl == -2 {
		r.ttl = 0
	}
	return r, nil
}

//go:embed fixedwindow.lua
var fwsrc string

// FixedWindow creates new counter which implements distributed counter using fixed window algorithm.
func FixedWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return newCounter(client, size, limit, fwsrc)
}

//go:embed slidingwindow.lua
var swsrc string

// SlidingWindow creates new counter which implements distributed counter using sliding window algorithm.
func SlidingWindow(client RedisClient, size time.Duration, limit uint) *Counter {
	return newCounter(client, size, limit, swsrc)
}

// Limiter implements distributed rate limiting.
type Limiter interface {
	// Limit applies the limit: increments key value of each distributed counter.
	Limit(ctx context.Context, key string) (Result, error)
}

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// NewLimiter creates new limiter which implements distributed rate limiting.
// Each limiter is created with pseudo-random name which may be set with options, every Redis key will be prefixed with this name.
// The rate of decreasing the window size on each next limiter call by default equal 1, may be set with options.
func NewLimiter(c *Counter, options ...func(*limiter)) Limiter {
	lt := &limiter{c, strconv.Itoa(random.Int()) + ":", 1}
	for _, option := range options {
		option(lt)
	}
	return lt
}

// WithLimiterName sets unique limiter name.
func WithLimiterName(name string) func(*limiter) {
	return func(lt *limiter) {
		lt.prefix = name + ":"
	}
}

// WithLimiterRate sets limiter rate of decreasing the window size on each next limiter call.
func WithLimiterRate(rate uint) func(*limiter) {
	return func(lt *limiter) {
		lt.rate = int(rate)
	}
}

// NewLimiterSuite creates new limiter suite which contains two or more limiters which run concurently on every limiter suite call.
func NewLimiterSuite(v1 Limiter, v2 Limiter, vs ...Limiter) Limiter {
	lts := append([]Limiter{v1, v2}, vs...)
	return &limiters{lts: lts, size: len(lts)}
}

type limiter struct {
	counter *Counter
	prefix  string
	rate    int
}

func (lt *limiter) Limit(ctx context.Context, key string) (Result, error) {
	return lt.counter.Count(ctx, lt.prefix+key, lt.rate)
}

type limiters struct {
	lts  []Limiter
	wg   sync.WaitGroup
	mu   sync.Mutex
	size int
}

const maxInt = int(^uint(0) >> 1)

func (ls *limiters) Limit(ctx context.Context, key string) (Result, error) {
	results := make([]result, ls.size)

	ls.mu.Lock()
	ls.wg.Add(ls.size)
	for i := 0; i < ls.size; i++ {
		go func(i int) {
			defer ls.wg.Done()
			r, err := ls.lts[i].Limit(ctx, key)
			results[i] = result{r, err}
		}(i)
	}
	ls.wg.Wait()
	ls.mu.Unlock()

	r := Result{0, int64(-1), maxInt}
	for i := 0; i < ls.size; i++ {
		v := results[i]
		if v.err != nil {
			return r, v.err
		}
		if v.result.OK() {
			if r.OK() && r.Remainder() > v.result.Remainder() { // minimal remainder
				r = v.result
			}
			continue
		}
		if r.OK() { // not ok first time
			r = v.result
			continue
		}
		if r.TTL() < v.result.TTL() { // maximum TTL
			r = v.result
		}
	}
	return r, nil
}

type result struct {
	result Result
	err    error
}
