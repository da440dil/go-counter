package counter

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type gwMock struct {
	mock.Mock
}

func (m *gwMock) Incr(key string, limit int64, ttl int64) (int64, error) {
	args := m.Called(key, limit, ttl)
	return args.Get(0).(int64), args.Error(1)
}

func TestCounter(t *testing.T) {
	const (
		key     = "key"
		limit   = int64(2)
		ttlTime = time.Millisecond * 500
		ttl     = int64(ttlTime / time.Millisecond)
	)

	params := Params{TTL: ttlTime, Limit: limit}

	t.Run("error", func(t *testing.T) {
		e := errors.New("any")
		gw := &gwMock{}
		gw.On("Incr", key, limit, ttl).Return(int64(-1), e)

		ctr := WithGateway(gw, params)

		err := ctr.Count(key)
		assert.Error(t, err)
		assert.Equal(t, e, err)
	})

	t.Run("failure", func(t *testing.T) {
		vErr := int64(42)
		gw := &gwMock{}
		gw.On("Incr", key, limit, ttl).Return(vErr, nil)

		ctr := WithGateway(gw, params)

		err := ctr.Count(key)
		assert.Error(t, err)
		assert.Exactly(t, newTTLError(vErr), err)
		gw.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		gw := &gwMock{}
		gw.On("Incr", key, limit, ttl).Return(int64(-1), nil)

		ctr := WithGateway(gw, params)

		err := ctr.Count(key)
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
	vErr := int64(42)
	err := newTTLError(vErr)
	assert.EqualError(t, err, errTooManyRequests.Error())
	assert.Equal(t, time.Duration(vErr)*time.Millisecond, err.TTL())
}
