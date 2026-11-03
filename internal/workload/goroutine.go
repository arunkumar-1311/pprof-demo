package workload

import (
	"context"
	"time"
)

// LeakGoroutine is the PROBLEM version: the goroutine waits on a channel that
// nobody will ever write to, so it can never exit.
// /debug/pprof/goroutine?debug=1 shows "N @ ..." with this function's stack.
func LeakGoroutine() {
	ch := make(chan int)
	go func() {
		<-ch // blocked forever
	}()
}

// SafeGoroutine is the FIXED version: the wait is also bound to a context, so
// the goroutine always exits when the timeout expires.
func SafeGoroutine(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	ch := make(chan int)
	go func() {
		defer cancel()
		select {
		case <-ch:
		case <-ctx.Done():
		}
	}()
}
