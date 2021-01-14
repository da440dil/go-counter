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
	c := &Counter{clientMock, size, limit, scr}
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
	require.Equal(t, msToDuration(-1), result.TTL())

	clientMock.AssertExpectations(t)
}

func msToDuration(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
