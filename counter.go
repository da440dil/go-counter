// Package counter provides functions for distributed rate limiting.
package counter

import (
	"errors"
	"time"

	gw "github.com/da440dil/go-counter/redis"
	"github.com/go-redis/redis"
)

// Gateway to storage to store a counter value.
type Gateway interface {
	// Incr sets key value and TTL of key if key not exists.
	// Increments key value if key exists.
	// Returns key value after increment, TTL of a key in milliseconds.
	Incr(key string, ttl int) (int, int, error)
}

// ErrInvalidTTL is the error returned when NewCounter receives invalid value of TTL.
var ErrInvalidTTL = errors.New("TTL must be greater than or equal to 1 millisecond")

// ErrInvalidLimit is the error returned when NewCounter receives invalid value of limit.
var ErrInvalidLimit = errors.New("Limit must be greater than zero")

// ErrInvaldKey is the error returned when key length is greater than 512 MB.
var ErrInvaldKey = errors.New("Key length must be less than or equal to 512 MB")

// Func is function returned by functions for setting options.
type Func func(c *Counter) error

// WithPrefix sets prefix of a key.
func WithPrefix(v string) Func {
	return func(c *Counter) error {
		if !isValidKey(v) {
			return ErrInvaldKey
		}
		c.prefix = v
		return nil
	}
}

// Counter implements distributed rate limiting.
type Counter struct {
	gateway Gateway
	ttl     int
	limit   int
	prefix  string
}

// NewCounterWithGateway creates new Counter using custom Gateway.
// Limit is maximum key value, must be greater than 0.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewCounterWithGateway(gateway Gateway, limit int, ttl time.Duration, options ...Func) (*Counter, error) {
	if limit < 1 {
		return nil, ErrInvalidLimit
	}
	if ttl < time.Millisecond {
		return nil, ErrInvalidTTL
	}
	c := &Counter{
		gateway: gateway,
		ttl:     durationToMilliseconds(ttl),
		limit:   limit,
	}
	for _, fn := range options {
		err := fn(c)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NewCounter creates new Counter using Redis Gateway.
// Limit is maximum key value, must be greater than 0.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewCounter(client *redis.Client, limit int, ttl time.Duration, options ...Func) (*Counter, error) {
	return NewCounterWithGateway(gw.NewGateway(client), limit, ttl, options...)
}

// Count increments key value.
// Returns limit remainder.
// Returns TTLError if limit exceeded.
func (c *Counter) Count(key string) (int, error) {
	key = c.prefix + key
	if !isValidKey(key) {
		return -1, ErrInvaldKey
	}
	value, ttl, err := c.gateway.Incr(key, c.ttl)
	if err != nil {
		return -1, err
	}
	rem := c.limit - value
	if rem < 0 {
		return rem, newTTLError(ttl)
	}
	return rem, nil
}

func durationToMilliseconds(duration time.Duration) int {
	return int(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}

// TTLError is the error returned when Counter failed to count.
type TTLError interface {
	Error() string
	TTL() time.Duration // Returns TTL of a key.
}

const ttlErrorMsg = "Too Many Requests"

type ttlError struct {
	ttl time.Duration
}

func newTTLError(ttl int) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return ttlErrorMsg
}

func (e *ttlError) TTL() time.Duration {
	return e.ttl
}

const maxKeyLen = 512000000

func isValidKey(key string) bool {
	return len([]byte(key)) <= maxKeyLen
}
