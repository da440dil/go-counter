package counter

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestNewLimiter(t *testing.T) {
	clientMock := &ClientMock{}
	size := time.Second
	limit := uint(100)
	sizev := int(size / time.Millisecond)
	limitv := int64(limit)

	v1 := NewLimiter(clientMock, WithLimit(size, limit, WithName("x")))
	require.Equal(t, &limiter{counter: &Counter{client: clientMock, script: fwscr, size: sizev, limit: limitv}, prefix: "x:", rate: 1}, v1)

	v2 := NewLimiter(clientMock, WithLimit(size, limit, WithName("x"), WithFixedWindow()))
	require.Equal(t, &limiter{counter: &Counter{client: clientMock, script: fwscr, size: sizev, limit: limitv}, prefix: "x:", rate: 1}, v2)

	v3 := NewLimiter(clientMock, WithLimit(size, limit, WithName("x"), WithSlidingWindow()))
	require.Equal(t, &limiter{counter: &Counter{client: clientMock, script: swscr, size: sizev, limit: limitv}, prefix: "x:", rate: 1}, v3)

	v4 := NewLimiter(clientMock, WithLimit(size, limit, WithName("x"), WithRate(2)))
	require.Equal(t, &limiter{counter: &Counter{client: clientMock, script: fwscr, size: sizev, limit: limitv}, prefix: "x:", rate: 2}, v4)

	v5 := NewLimiter(clientMock, WithLimit(size, limit, WithName("x")), WithLimit(size, limit, WithName("y")))
	require.Equal(t, &batchlimiter{client: clientMock, prefixes: []string{"x:", "y:"}, args: []interface{}{1, sizev, limitv, algFixed, 1, sizev, limitv, algFixed}}, v5)

	rnd := random
	random = rand.New(rand.NewSource(42))
	defer func() {
		random = rnd
	}()

	v6 := NewLimiter(clientMock, WithLimit(size, limit))
	require.Equal(t, &limiter{counter: &Counter{client: clientMock, script: fwscr, size: sizev, limit: limitv}, prefix: "3440579354231278675:", rate: 1}, v6)
}

func TestLimiter(t *testing.T) {
	clientMock := &ClientMock{}
	size := 1000
	limit := int64(100)
	c := &Counter{client: clientMock, script: fwscr, size: size, limit: limit}
	prefix := "x:"
	rate := 1
	lt := &limiter{counter: c, prefix: prefix, rate: rate}
	ctx := context.Background()
	hash := fwscr.Hash()

	var i interface{}

	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, hash, []string{"x:1"}, rate, size, limit).Return(redis.NewCmdResult(i, e))
	_, err := lt.Limit(ctx, "1")
	require.Equal(t, e, err)

	i = []interface{}{int64(1), int64(2), int64(100)}
	clientMock.On("EvalSha", ctx, hash, []string{"x:2"}, rate, size, limit).Return(redis.NewCmdResult(i, nil))
	result, err := lt.Limit(ctx, "2")
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, int64(2), result.Counter())
	require.Equal(t, limit-2, result.Remainder())
	require.Equal(t, msToDuration(100), result.TTL())

	clientMock.AssertExpectations(t)
}

func TestBatchLimiter(t *testing.T) {
	clientMock := &ClientMock{}
	rate := 1
	size := 1000
	limit := int64(100)
	prefixes := []string{"x:", "y:"}
	args := []interface{}{rate, size, limit, algFixed, rate, size, limit, algFixed}
	blt := &batchlimiter{client: clientMock, prefixes: prefixes, args: args}
	ctx := context.Background()
	hash := ltscr.Hash()

	var i interface{}

	e := errors.New("redis error")
	clientMock.On("EvalSha", ctx, hash, []string{"x:1", "y:1"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, e))
	_, err := blt.Limit(ctx, "1")
	require.Equal(t, e, err)

	clientMock.On("EvalSha", ctx, hash, []string{"x:2", "y:2"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "2")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{1, 2}
	clientMock.On("EvalSha", ctx, hash, []string{"x:3", "y:3"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "3")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{1, 2, 100}
	clientMock.On("EvalSha", ctx, hash, []string{"x:4", "y:4"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "4")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{1, 2, 100, 42}
	clientMock.On("EvalSha", ctx, hash, []string{"x:5", "y:5"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "5")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), 2, 100, 42}
	clientMock.On("EvalSha", ctx, hash, []string{"x:6", "y:6"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "6")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), int64(2), 100, 42}
	clientMock.On("EvalSha", ctx, hash, []string{"x:7", "y:7"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "7")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), int64(2), int64(100), 42}
	clientMock.On("EvalSha", ctx, hash, []string{"x:8", "y:8"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	_, err = blt.Limit(ctx, "8")
	require.Equal(t, ErrUnexpectedRedisResponse, err)

	i = []interface{}{int64(1), int64(2), int64(100), limit}
	clientMock.On("EvalSha", ctx, hash, []string{"x:9", "y:9"}, rate, size, limit, algFixed, rate, size, limit, algFixed).Return(redis.NewCmdResult(i, nil))
	result, err := blt.Limit(ctx, "9")
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, int64(2), result.Counter())
	require.Equal(t, limit-2, result.Remainder())
	require.Equal(t, msToDuration(100), result.TTL())

	clientMock.AssertExpectations(t)
}
