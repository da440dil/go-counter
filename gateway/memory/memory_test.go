package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const Key = "key"
const TTL = 100
const RefreshInterval = time.Millisecond * 20

func TestGateway(t *testing.T) {
	tt := millisecondsToDuration(TTL)
	timeout := millisecondsToDuration(TTL + 20)

	t.Run("set key value and TTL of key if key not exists", func(t *testing.T) {
		gw := New(RefreshInterval)

		v, ttl, err := gw.Incr(Key, TTL)
		assert.NoError(t, err)
		assert.Equal(t, 1, v)
		assert.Equal(t, TTL, ttl)

		item := gw.get(Key)
		assert.NotNil(t, item)
		assert.Equal(t, 1, item.value)
		diff := item.expiresAt.Sub(time.Now())
		assert.True(t, diff > 0 && diff <= tt)

		time.Sleep(timeout)

		item = gw.get(Key)
		assert.Nil(t, item)
	})

	t.Run("increment key value if key exists", func(t *testing.T) {
		gw := New(RefreshInterval)
		gw.Incr(Key, TTL)

		v, ttl, err := gw.Incr(Key, TTL)
		assert.NoError(t, err)
		assert.Equal(t, 2, v)
		assert.True(t, ttl > 0 && ttl <= TTL)

		item := gw.get(Key)
		assert.NotNil(t, item)
		assert.Equal(t, 2, item.value)
		diff := item.expiresAt.Sub(time.Now())
		assert.True(t, diff > 0 && diff <= tt)

		time.Sleep(timeout)

		item = gw.get(Key)
		assert.Nil(t, item)
	})
}

func BenchmarkGateway(b *testing.B) {
	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	testCases := []struct {
		ttl int
	}{
		{1000},
		{10000},
		{100000},
		{1000000},
	}

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("ttl %v", tc.ttl), func(b *testing.B) {
			gw := New(RefreshInterval)

			ttl := tc.ttl
			kl := len(keys)
			for i := 0; i < b.N; i++ {
				_, _, err := gw.Incr(keys[i%kl], ttl)
				assert.NoError(b, err)
			}
		})
	}
}
