package counter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ClientMock struct {
	mock.Mock
}

func (m *ClientMock) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	arg := m.Called(append([]interface{}{ctx, sha1, keys}, args...)...)
	return arg.Get(0).(*redis.Cmd)
}

func (m *ClientMock) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return nil
}

func (m *ClientMock) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	return nil
}

func (m *ClientMock) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return nil
}

func TestCounter(t *testing.T) {
	clientMock := &ClientMock{}
	size := 1000
	limit := 100
	scr := redis.NewScript("")
	c := &Counter{clientMock, scr, size, limit}
	ctx := context.Background()
	key := "key"
	keys := []string{key}

	var i interface{}

	v := 1
	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, e))
	_, err := c.Count(ctx, key, v)
	require.Equal(t, e, err)

	v = 2
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, key, v)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	v = 3
	i = []interface{}{1}
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, key, v)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	v = 4
	i = []interface{}{1, -1}
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, key, v)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	v = 5
	i = []interface{}{int64(1), -1}
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, key, v)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	v = 6
	i = []interface{}{int64(1), int64(-1)}
	clientMock.On("EvalSha", ctx, scr.Hash(), keys, v, size, limit).Return(redis.NewCmdResult(i, nil))
	result, err := c.Count(ctx, key, v)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, 1, result.Counter())
	require.Equal(t, limit-1, result.Remainder())
	require.Equal(t, msToDuration(-1), result.TTL())

	clientMock.AssertExpectations(t)
}

func TestLimiter(t *testing.T) {
	clientMock := &ClientMock{}
	size := 1000
	limit := 100
	scr := redis.NewScript("")
	c := &Counter{clientMock, scr, size, limit}
	ctx := context.Background()
	key := "key"

	rate := uint(1)
	v := int(rate)

	name := "name"
	lt := NewLimiter(c, WithLimiterName(name), WithLimiterRate(rate))

	clientMock.On("EvalSha", ctx, scr.Hash(), []string{name + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(rate), int64(-1)}, nil))
	result, err := lt.Limit(ctx, key)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, v, result.Counter())
	require.Equal(t, limit-v, result.Remainder())
	require.Equal(t, msToDuration(-1), result.TTL())

	clientMock.AssertExpectations(t)
}

func TestLimiterSuite(t *testing.T) {
	clientMock := &ClientMock{}
	size := 1000
	limit := 100
	scr := redis.NewScript("")
	c := &Counter{clientMock, scr, size, limit}
	ctx := context.Background()
	key := "key"

	rate := uint(1)
	v := int(rate)

	n1 := "name1"
	lt1 := NewLimiter(c, WithLimiterName(n1), WithLimiterRate(rate))
	r1 := 25
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n1 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r1), int64(-1)}, nil))

	n2 := "name2"
	lt2 := NewLimiter(c, WithLimiterName(n2), WithLimiterRate(rate))
	r2 := 58
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n2 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r2), int64(-1)}, nil))

	n3 := "name3"
	lt3 := NewLimiter(c, WithLimiterName(n3), WithLimiterRate(rate))
	r3 := 26
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n3 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r3), int64(-1)}, nil))

	ls1 := NewLimiterSuite(lt1, lt2, lt3)
	result, err := ls1.Limit(ctx, key)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, r2, result.Counter())
	require.Equal(t, limit-r2, result.Remainder())
	require.Equal(t, msToDuration(-1), result.TTL())

	n4 := "name4"
	lt4 := NewLimiter(c, WithLimiterName(n4), WithLimiterRate(rate))
	r4 := 58
	t4 := int64(42)
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n4 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r4), t4}, nil))

	ls2 := NewLimiterSuite(lt1, lt4, lt3)
	result, err = ls2.Limit(ctx, key)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, r4, result.Counter())
	require.Equal(t, limit-r4, result.Remainder())
	require.Equal(t, msToDuration(t4), result.TTL())

	n5 := "name5"
	lt5 := NewLimiter(c, WithLimiterName(n5), WithLimiterRate(rate))
	r5 := 58
	t5 := int64(42)
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n5 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r5), t5}, nil))

	n6 := "name6"
	lt6 := NewLimiter(c, WithLimiterName(n6), WithLimiterRate(rate))
	r6 := 25
	t6 := int64(75)
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n6 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r6), t6}, nil))

	n7 := "name7"
	lt7 := NewLimiter(c, WithLimiterName(n7), WithLimiterRate(rate))
	r7 := 26
	t7 := int64(74)
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n7 + ":" + key}, v, size, limit).Return(redis.NewCmdResult([]interface{}{int64(r7), t7}, nil))

	ls3 := NewLimiterSuite(lt5, lt6, lt7)
	result, err = ls3.Limit(ctx, key)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, r6, result.Counter())
	require.Equal(t, limit-r6, result.Remainder())
	require.Equal(t, msToDuration(t6), result.TTL())

	n8 := "name8"
	lt8 := NewLimiter(c, WithLimiterName(n8), WithLimiterRate(rate))
	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, scr.Hash(), []string{n8 + ":" + key}, v, size, limit).Return(redis.NewCmdResult(0, e))

	ls4 := NewLimiterSuite(lt1, lt8, lt2)
	_, err = ls4.Limit(ctx, key)
	require.Equal(t, e, err)

	clientMock.AssertExpectations(t)
}

func msToDuration(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
