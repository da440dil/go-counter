// Package memory implements Gateway to memory storage to store a counter value.
package memory

import (
	"runtime"
	"sync"
	"time"
)

// Gateway is a gateway to memory storage.
type Gateway struct {
	*storage
}

// New creates new Gateway.
func New(cleanupInterval time.Duration) *Gateway {
	s := newStorage()
	// This trick ensures that the janitor goroutine does not keep
	// the returned Gateway object from being garbage collected.
	// When it is garbage collected, the finalizer stops the janitor goroutine,
	// after which storage can be collected.
	G := &Gateway{s}
	runJanitor(s, cleanupInterval)
	runtime.SetFinalizer(G, stopJanitor)
	return G
}

type item struct {
	value     int
	expiresAt time.Time
}

type storage struct {
	items   map[string]*item
	mutex   sync.Mutex
	janitor *janitor
}

func newStorage() *storage {
	s := &storage{}
	s.init()
	return s
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
			return v.value, durationToMilliseconds(exp), nil
		}
	}
	s.items[key] = &item{
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

func (s *storage) get(key string) *item {
	v, ok := s.items[key]
	if ok {
		return v
	}
	return nil
}

func (s *storage) init() {
	s.mutex.Lock()
	s.items = make(map[string]*item)
	s.mutex.Unlock()
}

func durationToMilliseconds(duration time.Duration) int {
	return int(duration / time.Millisecond)
}

func millisecondsToDuration(ttl int) time.Duration {
	return time.Duration(ttl) * time.Millisecond
}

type janitor struct {
	interval time.Duration
	stop     chan struct{}
}

func (j *janitor) Run(c *storage) {
	ticker := time.NewTicker(j.interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(gw *Gateway) {
	close(gw.janitor.stop)
}

func runJanitor(s *storage, interval time.Duration) {
	j := &janitor{
		interval: interval,
		stop:     make(chan struct{}),
	}
	s.janitor = j
	go j.Run(s)
}
