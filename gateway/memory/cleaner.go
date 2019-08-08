package memory

import "time"

type cleaner struct {
	interval time.Duration
	stop     chan struct{}
}

func newCleaner(interval time.Duration) *cleaner {
	return &cleaner{
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (c *cleaner) Run(fn func()) {
	ticker := time.NewTicker(c.interval)
	for {
		select {
		case <-ticker.C:
			fn()
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
}

func (c *cleaner) Stop() {
	close(c.stop)
}
