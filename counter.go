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
	// Incr sets key value and ttl of key if key not exists.
	// Increments key value if key exists.
	// Returns -1 if key value less than or equal limit.
	// Returns ttl in milliseconds if key value greater than limit.
	Incr(key string, limit int64, ttl int64) (int64, error)
}

// Params defines parameters for creating new Counter.
type Params struct {
	TTL    time.Duration // TTL of a key. Must be greater than or equal to 1 millisecond.
	Limit  int64         // Maximum key value. Must be greater than 0.
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
		limit:   params.Limit,
		ttl:     durationToMilliseconds(params.TTL),
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
	limit   int64
	ttl     int64
	prefix  string
}

// Count increments key value. Returns TTLError if limit exceeded.
func (c *Counter) Count(key string) error {
	ttl, err := c.gateway.Incr(c.prefix+key, c.limit, c.ttl)
	if err != nil {
		return err
	}
	if ttl != -1 {
		return newTTLError(ttl)
	}
	return nil
}

func durationToMilliseconds(duration time.Duration) int64 {
	return int64(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int64) time.Duration {
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

func newTTLError(ttl int64) *ttlError {
	return &ttlError{millisecondsToDuration(ttl)}
}

func (e *ttlError) Error() string {
	return errTooManyRequests.Error()
}

func (e *ttlError) TTL() time.Duration {
	return e.ttl
}
