package counter

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestSlidingWindow(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	require.NoError(t, err)

	size := time.Second
	counter := SlidingWindow(client, size, 100)

	result, err := counter.Count(ctx, key, 101)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, int64(0), result.Counter())
	require.Equal(t, int64(100), result.Remainder())
	require.True(t, result.TTL() >= msToDuration(0) && result.TTL() <= size)

	time.Sleep(result.TTL()) // wait for the next window to start

	result, err = counter.Count(ctx, key, 20)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, int64(20), result.Counter())
	require.Equal(t, int64(80), result.Remainder())
	require.Equal(t, msToDuration(-1), result.TTL())

	result, err = counter.Count(ctx, key, 30)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, int64(50), result.Counter())
	require.Equal(t, int64(50), result.Remainder())
	require.Equal(t, msToDuration(-1), result.TTL())

	result, err = counter.Count(ctx, key, 51)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, int64(50), result.Counter())
	require.Equal(t, int64(50), result.Remainder())
	require.True(t, result.TTL() >= msToDuration(0) && result.TTL() <= size)

	time.Sleep(result.TTL()) // wait for the next window to start

	result, err = counter.Count(ctx, key, 70)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.True(t, result.Counter() > 30 && result.Counter() <= 100)
	require.True(t, result.Remainder() >= 0 && result.Remainder() <= 70)
	require.True(t, result.TTL() >= msToDuration(0) && result.TTL() <= size)

	time.Sleep(msToDuration(700)) // wait for the most time of the current window to pass

	result, err = counter.Count(ctx, key, 70)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.True(t, result.Counter() > 70 && result.Counter() <= 100)
	require.True(t, result.Remainder() >= 0 && result.Remainder() <= 30)
	require.Equal(t, msToDuration(-1), result.TTL())
}
