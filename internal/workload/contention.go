package workload

import (
	"sync"
	"sync/atomic"
	"time"
)

// SlowCounter is the PROBLEM version: it does slow work while holding the lock,
// so every request queues behind the previous one.
// The mutex profile points at the Lock() call inside Inc.
type SlowCounter struct {
	mu   sync.Mutex
	n    int
	work time.Duration
}

func NewSlowCounter(work time.Duration) *SlowCounter {
	return &SlowCounter{work: work}
}

func (c *SlowCounter) Inc() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	time.Sleep(c.work) // simulated I/O under the lock
	c.n++
	return c.n
}

// FastCounter is the FIXED version: slow work happens outside any lock and the
// shared counter is an atomic.
type FastCounter struct {
	n    atomic.Int64
	work time.Duration
}

func NewFastCounter(work time.Duration) *FastCounter {
	return &FastCounter{work: work}
}

func (c *FastCounter) Inc() int64 {
	time.Sleep(c.work) // same work, no lock held
	return c.n.Add(1)
}
