package counter

import "time"

type counterError string

func (e counterError) Error() string {
	return string(e)
}

// ErrInvalidTTL is the error returned when NewCounter receives invalid value of TTL.
const ErrInvalidTTL = counterError("counter: TTL must be greater than or equal to 1 millisecond")

// ErrInvalidLimit is the error returned when NewCounter receives invalid value of limit.
const ErrInvalidLimit = counterError("counter: limit must be greater than 0")

// ErrInvalidKey is the error returned when key size with prefix is greater than 512 MB.
const ErrInvalidKey = counterError("counter: key size with prefix must be less than or equal to 512 MB")

// ErrTooManyRequests is the error wrapped with TTLError.
const ErrTooManyRequests = counterError("counter: too many requests")

// TTLError is the error returned when Counter failed to count.
type TTLError struct {
	err error
	ttl time.Duration
}

func newTTLError(ttl int) *TTLError {
	return &TTLError{
		err: ErrTooManyRequests,
		ttl: millisecondsToDuration(ttl),
	}
}

func (e *TTLError) Error() string {
	return e.err.Error()
}

// TTL returns TTL of a key.
func (e *TTLError) TTL() time.Duration {
	return e.ttl
}

func (e *TTLError) Unwrap() error {
	return e.err
}
