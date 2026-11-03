// Package debugserver runs the pprof / expvar endpoints on their own port.
//
// Why a separate server?
//   - Gin does not use http.DefaultServeMux, so the blank import of
//     net/http/pprof would not be reachable through the Gin router.
//   - /debug/pprof must never be public. Binding to localhost keeps it private
//     (in Kubernetes, reach it with `kubectl port-forward`).
package debugserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on http.DefaultServeMux
	"runtime"
	"time"

	_ "expvar" // registers /debug/vars on http.DefaultServeMux
)

// Start enables the optional profiles and serves diagnostics on addr.
//
// NOTE: no WriteTimeout is set on purpose. /debug/pprof/profile?seconds=30 and
// /debug/pprof/trace?seconds=N hold the response open for that long, and a
// short WriteTimeout would cut them off.
func Start(addr string, mutexFraction, blockRateNs int) *http.Server {
	// Mutex and block profiles are OFF by default in Go and must be switched on.
	runtime.SetMutexProfileFraction(mutexFraction)
	runtime.SetBlockProfileRate(blockRateNs)

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("[debug] pprof on http://%s/debug/pprof/", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[debug] server stopped: %v", err)
		}
	}()
	return srv
}

// LogRuntimeStats prints goroutine count and heap size on an interval.
// A goroutine count that only ever grows is the quickest leak signal.
func LogRuntimeStats(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("[stats] goroutines=%d heap_alloc=%dMB heap_objects=%d num_gc=%d gc_pause_total=%s",
				runtime.NumGoroutine(), m.HeapAlloc>>20, m.HeapObjects, m.NumGC,
				time.Duration(m.PauseTotalNs))
		}
	}
}
