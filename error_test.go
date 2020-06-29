package counter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterError(t *testing.T) {
	v := "any"
	err := counterError(v)
	assert.Equal(t, v, err.Error())
}

func TestTTLError(t *testing.T) {
	et := 42
	err := newTTLError(et)
	assert.True(t, errors.Is(err, ErrTooManyRequests))
	assert.Equal(t, ErrTooManyRequests.Error(), err.Error())
	assert.Equal(t, millisecondsToDuration(et), err.TTL())
}
