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
	limit := int64(100)
	c := &Counter{client: clientMock, script: fwscr, size: size, limit: limit}
	ctx := context.Background()
	hash := fwscr.Hash()
	value := 1

	var i interface{}

	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, hash, []string{"1"}, value, size, limit).Return(redis.NewCmdResult(i, e))
	_, err := c.Count(ctx, "1", value)
	require.Equal(t, e, err)

	clientMock.On("EvalSha", ctx, hash, []string{"2"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, "2", value)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{1, 2}
	clientMock.On("EvalSha", ctx, hash, []string{"3"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, "3", value)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{1, 2, 100}
	clientMock.On("EvalSha", ctx, hash, []string{"4"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, "4", value)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), 2, 100}
	clientMock.On("EvalSha", ctx, hash, []string{"5"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, "5", value)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), int64(2), 100}
	clientMock.On("EvalSha", ctx, hash, []string{"6"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	_, err = c.Count(ctx, "6", value)
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), int64(2), int64(100)}
	clientMock.On("EvalSha", ctx, hash, []string{"7"}, value, size, limit).Return(redis.NewCmdResult(i, nil))
	result, err := c.Count(ctx, "7", value)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, int64(2), result.Counter())
	require.Equal(t, limit-2, result.Remainder())
	require.Equal(t, msToDuration(100), result.TTL())

	clientMock.AssertExpectations(t)
}

func msToDuration(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
