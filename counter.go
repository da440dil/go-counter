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

// Params defines parameters for creating new Counter.
type Params struct {
	TTL    time.Duration // TTL of a key. Must be greater than or equal to 1 millisecond.
	Limit  int           // Maximum key value. Must be greater than 0.
	Prefix string        // Prefix of a key. Optional.
}

var errInvalidTTL = errors.New("TTL must be greater than or equal to 1 millisecond")
var errInvalidLimit = errors.New("Limit must be greater than zero")

func (p Params) validate() {
	if p.TTL < time.Millisecond {
		panic(errInvalidTTL)
	}
	if p.Limit < 1 {
		panic(errInvalidLimit)
	}
}

// WithGateway creates new Counter using custom Gateway.
func WithGateway(gateway Gateway, params Params) *Counter {
	params.validate()
	return &Counter{
		gateway: gateway,
		ttl:     durationToMilliseconds(params.TTL),
		limit:   params.Limit,
		prefix:  params.Prefix,
	}
}

// NewCounter creates new Counter using Redis Gateway.
func NewCounter(client *redis.Client, params Params) *Counter {
	return WithGateway(gw.NewGateway(client), params)
}

// Counter implements distributed rate limiting.
type Counter struct {
	gateway Gateway
	ttl     int
	limit   int
	prefix  string
}

// Count increments key value.
// Returns limit remainder.
// Returns TTLError if limit exceeded.
func (c *Counter) Count(key string) (int, error) {
	value, ttl, err := c.gateway.Incr(c.prefix+key, c.ttl)
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

var errTooManyRequests = errors.New("Too Many Requests")

type ttlError struct {
	ttl time.Duration
}

func newTTLError(ttl int) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return errTooManyRequests.Error()
}

func (e *ttlError) TTL() time.Duration {
	return e.ttl
}
