package counter

import (
	"errors"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type gwMock struct {
	mock.Mock
}

func (m *gwMock) Incr(key string, ttl int) (int, int, error) {
	args := m.Called(key, ttl)
	return args.Int(0), args.Int(1), args.Error(2)
}

const Addr = "localhost:6379"
const DB = 10

const Key = "key"
const TTL = time.Millisecond * 100
const Limit = 1

var p = make([]byte, MaxKeySize+1)
var invalidKey = *(*string)(unsafe.Pointer(&p))

func TestNewCounter(t *testing.T) {
	gw := &gwMock{}

	t.Run("ErrInvalidLimit", func(t *testing.T) {
		_, err := New(0, time.Microsecond, WithGateway(gw))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidLimit, err)
	})

	t.Run("ErrInvalidTTL", func(t *testing.T) {
		_, err := New(Limit, time.Microsecond, WithGateway(gw))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidTTL, err)
	})

	t.Run("success", func(t *testing.T) {
		c, err := New(Limit, TTL, WithGateway(gw))
		assert.NoError(t, err)
		assert.IsType(t, &Counter{}, c)
	})
}

func TestOptions(t *testing.T) {
	gw := &gwMock{}

	t.Run("ErrInvaldKey", func(t *testing.T) {
		_, err := New(Limit, TTL, WithPrefix(invalidKey), WithGateway(gw))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	t.Run("success", func(t *testing.T) {
		c, err := New(Limit, TTL, WithPrefix(""), WithGateway(gw))
		assert.NoError(t, err)
		assert.IsType(t, &Counter{}, c)
	})
}

func TestCounter(t *testing.T) {
	ttl := int(TTL / time.Millisecond)

	t.Run("ErrInvaldKey", func(t *testing.T) {
		gw := &gwMock{}

		c, err := New(Limit, TTL, WithGateway(gw))
		assert.NoError(t, err)

		v, err := c.Count(invalidKey)
		assert.Equal(t, -1, v)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKey, err)
	})

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(-1, 42, e)

		c, err := New(Limit, TTL, WithGateway(gw))
		assert.NoError(t, err)

		v, err := c.Count(Key)
		assert.Equal(t, -1, v)
		assert.Error(t, err)
		assert.Equal(t, e, err)
		gw.AssertExpectations(t)
	})

	t.Run("failure", func(t *testing.T) {
		et := 42
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(Limit+1, et, nil)

		c, err := New(Limit, TTL, WithGateway(gw))
		assert.NoError(t, err)

		v, err := c.Count(Key)
		assert.Equal(t, -1, v)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(et), err)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(Limit, 42, nil)

		c, err := New(Limit, TTL, WithGateway(gw))
		assert.NoError(t, err)

		v, err := c.Count(Key)
		assert.Equal(t, 0, v)
		assert.NoError(t, err)
		gw.AssertExpectations(t)
	})
}

func TestCounterDefaultGateway(t *testing.T) {
	c, err := New(Limit, TTL)
	assert.NoError(t, err)
	assert.IsType(t, &Counter{}, c)
	assert.NotNil(t, c.gateway)
}
