package counter

import (
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis"
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

func TestNewCounter(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: Addr, DB: DB})
	defer client.Close()

	ctr := NewCounter(client, Params{TTL: TTL, Limit: Limit})
	assert.IsType(t, &Counter{}, ctr)
}

func TestCounter(t *testing.T) {
	params := Params{TTL: TTL, Limit: Limit}

	ttl := durationToMilliseconds(TTL)

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(-1, 42, e)

		ctr := WithGateway(gw, params)

		v, err := ctr.Count(Key)
		assert.Equal(t, -1, v)
		assert.Error(t, err)
		assert.Equal(t, e, err)
		gw.AssertExpectations(t)
	})

	t.Run("failure", func(t *testing.T) {
		et := 42
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(Limit+1, et, nil)

		ctr := WithGateway(gw, params)

		v, err := ctr.Count(Key)
		assert.Equal(t, -1, v)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(et), err)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Incr", Key, ttl).Return(Limit, 42, nil)

		ctr := WithGateway(gw, params)

		v, err := ctr.Count(Key)
		assert.Equal(t, 0, v)
		assert.NoError(t, err)
		gw.AssertExpectations(t)
	})
}

func TestParams(t *testing.T) {
	t.Run("invalid ttl", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidTTL, err)
		}()

		Params{TTL: time.Microsecond}.validate()
	})

	t.Run("invalid limit", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			err, ok := r.(error)
			assert.True(t, ok)
			assert.Error(t, err)
			assert.Equal(t, errInvalidLimit, err)
		}()

		Params{TTL: time.Millisecond}.validate()
	})
}

func TestTTLError(t *testing.T) {
	et := 42
	err := newTTLError(et)
	assert.EqualError(t, err, errTooManyRequests.Error())
	assert.Equal(t, millisecondsToDuration(et), err.TTL())
}
