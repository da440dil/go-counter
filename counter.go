// Package counter provides functions for distributed rate limiting.
package counter

import (
	"time"

	gw "github.com/da440dil/go-counter/gateway/memory"
)

// Gateway to storage to store a counter value.
type Gateway interface {
	// Incr sets key value and TTL of key if key not exists.
	// Increments key value if key exists.
	// Returns key value after increment.
	// Returns TTL of a key in milliseconds.
	Incr(key string, ttl int) (int, int, error)
}

type counterError string

func (e counterError) Error() string {
	return string(e)
}

// ErrInvalidTTL is the error returned when NewCounter receives invalid value of TTL.
const ErrInvalidTTL = counterError("counter: TTL must be greater than or equal to 1 millisecond")

// ErrInvalidLimit is the error returned when NewCounter receives invalid value of limit.
const ErrInvalidLimit = counterError("counter: limit must be greater than 0")

// ErrInvalidKey is the error returned when key size is greater than 512 MB.
const ErrInvalidKey = counterError("counter: key size must be less than or equal to 512 MB")

// Option is function returned by functions for setting options.
type Option func(c *Counter) error

// WithGateway sets counter gateway.
// Gateway is gateway to storage to store a counter value.
// If gateway not set counter creates new memory gateway
// with expired keys cleanup every 100 milliseconds.
func WithGateway(v Gateway) Option {
	return func(c *Counter) error {
		c.gateway = v
		return nil
	}
}

// WithPrefix sets prefix of a key.
func WithPrefix(v string) Option {
	return func(c *Counter) error {
		if !isValidKey(v) {
			return ErrInvalidKey
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

// New creates new Counter.
// Limit is maximum key value, must be greater than 0.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func New(limit int, ttl time.Duration, options ...Option) (*Counter, error) {
	if limit < 1 {
		return nil, ErrInvalidLimit
	}
	if ttl < time.Millisecond {
		return nil, ErrInvalidTTL
	}
	c := &Counter{
		ttl:   durationToMilliseconds(ttl),
		limit: limit,
	}
	for _, fn := range options {
		if err := fn(c); err != nil {
			return nil, err
		}
	}
	if c.gateway == nil {
		c.gateway = gw.New(time.Millisecond * 100)
	}
	return c, nil
}

// Count increments key value.
// Returns limit remainder.
// Returns TTLError if limit exceeded.
func (c *Counter) Count(key string) (int, error) {
	key = c.prefix + key
	if !isValidKey(key) {
		return -1, ErrInvalidKey
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

const ttlErrorMsg = "counter: too many requests"

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

// MaxKeySize is maximum key size in bytes.
const MaxKeySize = 512000000

func isValidKey(key string) bool {
	return len([]byte(key)) <= MaxKeySize
}
