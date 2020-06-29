// Package memory implements Gateway to memory storage to store a counter value.
package memory

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/da440dil/go-ticker"
)

// Gateway to memory storage.
type Gateway struct {
	*storage
}

// New creates new Gateway.
func New(cleanupInterval time.Duration) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	s := &storage{
		items:  map[string]item{},
		cancel: cancel,
	}
	gw := &Gateway{s}
	go ticker.Run(ctx, s.deleteExpired, cleanupInterval)
	runtime.SetFinalizer(gw, finalizer)
	return gw
}

func finalizer(gw *Gateway) {
	gw.cancel()
}

type item struct {
	value     int
	expiresAt time.Time
}

type storage struct {
	items  map[string]item
	mutex  sync.Mutex
	cancel func()
}

func (s *storage) Incr(key string, ttl int) (int, int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	v, ok := s.items[key]
	if ok {
		exp := v.expiresAt.Sub(now)
		if exp > 0 {
			v.value++
			s.items[key] = v
			return v.value, durationToMilliseconds(exp), nil
		}
	}
	s.items[key] = item{
		value:     1,
		expiresAt: now.Add(millisecondsToDuration(ttl)),
	}
	return 1, ttl, nil
}

func (s *storage) deleteExpired() {
	s.mutex.Lock()

	now := time.Now()
	for k, v := range s.items {
		exp := v.expiresAt.Sub(now)
		if exp <= 0 {
			delete(s.items, k)
		}
	}

	s.mutex.Unlock()
}

func durationToMilliseconds(duration time.Duration) int {
	return int(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}
