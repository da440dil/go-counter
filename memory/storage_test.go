package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	const (
		key   = "key"
		limit = 2
		ttl   = time.Millisecond * 1000
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(ttl)

	var err error
	var v int64

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	delete(storage.db, key)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	storage.cancel()
}

func TestTTL(t *testing.T) {
	const (
		key   = "key"
		limit = 2
		ttl   = time.Millisecond * 100
		ms    = int64(ttl / time.Millisecond)
	)

	storage := NewStorage(ttl)

	var err error
	var v int64

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v >= 0 && v <= ms)

	time.Sleep(time.Millisecond * 200)

	v, err = storage.Incr(key, limit, ttl)
	assert.NoError(t, err)
	assert.True(t, v == -1)

	storage.cancel()
}
