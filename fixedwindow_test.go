package counter

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestFixedWindow(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	defer client.Close()

	ctx := context.Background()
	key := "key"
	err := client.Del(ctx, key).Err()
	require.NoError(t, err)

	size := time.Second
	counter := FixedWindow(client, size, 100)

	result, err := counter.Count(ctx, key, 101)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, 0, result.Counter())
	require.Equal(t, msToDuration(-2), result.TTL())

	result, err = counter.Count(ctx, key, 20)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, 20, result.Counter())
	require.Equal(t, msToDuration(-1), result.TTL())

	result, err = counter.Count(ctx, key, 30)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, 50, result.Counter())
	require.Equal(t, msToDuration(-1), result.TTL())

	result, err = counter.Count(ctx, key, 51)
	require.NoError(t, err)
	require.False(t, result.OK())
	require.Equal(t, 50, result.Counter())
	require.True(t, result.TTL() >= msToDuration(0) && result.TTL() <= size)

	time.Sleep(result.TTL()) // wait for the next window to start

	result, err = counter.Count(ctx, key, 70)
	require.NoError(t, err)
	require.True(t, result.OK())
	require.Equal(t, 70, result.Counter())
	require.Equal(t, msToDuration(-1), result.TTL())
}
