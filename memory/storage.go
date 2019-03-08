// Package memory is for creating storage in memory.
package memory

import (
	"context"
	"sync"
	"time"
)

// NewStorage allocates and returns new Storage.
func NewStorage(refreshInterval time.Duration) *Storage {
	ctx, cancel := context.WithCancel(context.Background())
	storage := &Storage{
		db:      make(map[string]*data),
		timeout: refreshInterval,
		done:    ctx.Done(),
		cancel:  cancel,
	}
	go storage.init()
	return storage
}

// Storage implements storage in memory.
type Storage struct {
	db      map[string]*data
	timeout time.Duration
	mutex   sync.Mutex
	done    <-chan struct{}
	cancel  context.CancelFunc
}

type data struct {
	value uint64
	ttl   time.Duration
}

func (s *Storage) init() {
	timer := time.NewTimer(s.timeout)
	defer timer.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-timer.C:
		}

		s.mutex.Lock()

		for k, v := range s.db {
			v.ttl = v.ttl - s.timeout
			if v.ttl <= 0 {
				delete(s.db, k)
			}
		}

		s.mutex.Unlock()

		timer.Reset(s.timeout)
	}
}

func (s *Storage) Incr(key string, limit uint64, ttl time.Duration) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.db[key]
	if ok {
		v.value = v.value + 1
		if v.value > limit {
			return int64(v.ttl / time.Millisecond), nil
		}
		return -1, nil
	}
	s.db[key] = &data{value: 1, ttl: ttl}
	return -1, nil
}
