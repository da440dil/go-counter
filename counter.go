// Package counter contains functions and data structures for distributed rate limiting.
package counter

import "time"

// Storage implements key value storage.
type Storage interface {
	// Incr sets key value and ttl of key if key not exists or increment key value if key exists,
	// returns -1 if key value less than or equal limit,
	// returns ttl in milliseconds if key value greater than limit.
	Incr(key string, limit uint64, ttl time.Duration) (int64, error)
}

// Params defines parameters for creating new Counter.
type Params struct {
	TTL    time.Duration // TTL of key (required).
	Limit  uint64        // Maximum key value (optional, should be greater than 0, by default equals 1).
	Prefix string        // Prefix of key (optional).
}

// NewCounter allocates and returns new Counter.
func NewCounter(storage Storage, params Params) *Counter {
	var limit uint64 = 1
	if params.Limit > 1 {
		limit = params.Limit
	}
	return &Counter{
		storage: storage,
		limit:   limit,
		ttl:     params.TTL,
		prefix:  params.Prefix,
	}
}

// Counter implements distributed rate limiting.
type Counter struct {
	storage Storage
	limit   uint64
	ttl     time.Duration
	prefix  string
}

// Count increments key value,
// returns -1 if key value less than or equal limit,
// returns ttl in milliseconds if key value greater than limit.
func (c *Counter) Count(key string) (int64, error) {
	return c.storage.Incr(c.prefix+key, c.limit, c.ttl)
}
