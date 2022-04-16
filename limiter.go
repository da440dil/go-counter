package counter

import (
	"context"
	_ "embed"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// Limiter implements distributed rate limiting.
type Limiter interface {
	// Limit applies the limit.
	Limit(ctx context.Context, key string) (Result, error)
}

type params struct {
	prefix string
	alg    int
	rate   int
	size   int
	limit  int64
}

const (
	algFixed   = 1
	algSliding = 2
)

// WithLimit creates parameters to build a limit.
//
// By default a limit uses fixed window algorithm, may be set with options.
// Each limit is created with pseudo-random name which may be set with options.
// The rate of decreasing the window size on each next application of the limit by default equal 1, may be set with options.
func WithLimit(size time.Duration, limit uint, options ...func(*params)) *params {
	p := &params{alg: algFixed, size: int(size / time.Millisecond), limit: int64(limit)}
	for _, opt := range options {
		opt(p)
	}
	if p.prefix == "" {
		p.prefix = strconv.Itoa(random.Int()) + ":"
	}
	if p.rate == 0 {
		p.rate = 1
	}
	return p
}

// WithFixedWindow sets fixed window algorithm for the limit.
func WithFixedWindow() func(*params) {
	return func(p *params) {
		p.alg = algFixed
	}
}

// WithSlidingWindow sets sliding window algorithm for the limit.
func WithSlidingWindow() func(*params) {
	return func(p *params) {
		p.alg = algSliding
	}
}

// WithName sets unique name for the limit, every Redis key will be prefixed with this name.
func WithName(name string) func(*params) {
	return func(p *params) {
		p.prefix = name + ":"
	}
}

// WithRate sets the rate of decreasing the window size on each next application of the limit.
func WithRate(rate uint) func(*params) {
	return func(p *params) {
		p.rate = int(rate)
	}
}

// NewLimiter creates new limiter which implements distributed rate limiting.
func NewLimiter(client RedisClient, first *params, rest ...*params) Limiter {
	n := len(rest)
	if n == 0 {
		var scr *redis.Script
		if first.alg == algFixed {
			scr = fwscr
		} else {
			scr = swscr
		}
		c := &Counter{client: client, script: scr, size: first.size, limit: first.limit}
		return &limiter{counter: c, prefix: first.prefix, rate: first.rate}
	}

	size := n + 1
	prefixes := make([]string, size)
	prefixes[0] = first.prefix
	args := make([]interface{}, size*4)
	args[0] = first.rate
	args[1] = first.size
	args[2] = first.limit
	args[3] = first.alg

	z := 0
	for i := 0; i < n; i++ {
		z += 4
		prefixes[i+1] = rest[i].prefix
		args[z] = rest[i].rate
		args[z+1] = rest[i].size
		args[z+2] = rest[i].limit
		args[z+3] = rest[i].alg
	}

	return &batchlimiter{client: client, prefixes: prefixes, args: args}
}

type limiter struct {
	counter *Counter
	prefix  string
	rate    int
}

func (lt *limiter) Limit(ctx context.Context, key string) (Result, error) {
	return lt.counter.Count(ctx, lt.prefix+key, lt.rate)
}

type batchlimiter struct {
	client   RedisClient
	prefixes []string
	args     []interface{}
}

//go:embed limit.lua
var ltsrc string
var ltscr = redis.NewScript(ltsrc)

func (blt *batchlimiter) Limit(ctx context.Context, key string) (Result, error) {
	keys := make([]string, len(blt.prefixes))
	for i := 0; i < len(blt.prefixes); i++ {
		keys[i] = blt.prefixes[i] + key
	}
	r := Result{}
	res, err := ltscr.Run(ctx, blt.client, keys, blt.args...).Result()
	if err != nil {
		return r, err
	}
	arr, ok := res.([]interface{})
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	if len(arr) != 4 {
		return r, ErrUnexpectedRedisResponse
	}
	r.ok, ok = arr[0].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	r.counter, ok = arr[1].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	r.ttl, ok = arr[2].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	r.limit, ok = arr[3].(int64)
	if !ok {
		return r, ErrUnexpectedRedisResponse
	}
	return r, nil
}
