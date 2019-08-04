// Package counter provides functions for distributed rate limiting.
package counter

import (
	"errors"
	"time"
)

// Gateway to storage to store a counter value.
type Gateway interface {
	// Incr sets key value and TTL of key if key not exists.
	// Increments key value if key exists.
	// Returns key value after increment.
	// Returns TTL of a key in milliseconds.
	Incr(key string, ttl int) (int, int, error)
}

// ErrInvalidTTL is the error returned when NewCounter receives invalid value of TTL.
var ErrInvalidTTL = errors.New("TTL must be greater than or equal to 1 millisecond")

// ErrInvalidLimit is the error returned when NewCounter receives invalid value of limit.
var ErrInvalidLimit = errors.New("Limit must be greater than zero")

// ErrInvalidKey is the error returned when key size is greater than 512 MB.
var ErrInvalidKey = errors.New("Key size must be less than or equal to 512 MB")

// Option is function returned by functions for setting options.
type Option func(c *Counter) error

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

// NewCounter creates new Counter.
// Gateway is gateway to storage to store a counter value.
// Limit is maximum key value, must be greater than 0.
// TTL is TTL of a key, must be greater than or equal to 1 millisecond.
// Options are functional options.
func NewCounter(gateway Gateway, limit int, ttl time.Duration, options ...Option) (*Counter, error) {
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

// MaxKeySize is maximum key size in bytes.
const MaxKeySize = 512000000

func isValidKey(key string) bool {
	return len([]byte(key)) <= MaxKeySize
}
