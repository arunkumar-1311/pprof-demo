// Package handlers exposes one HTTP endpoint per profiling scenario.
//
// Every endpoint accepts ?fixed=true to run the corrected implementation, so
// you can profile "before", flip the flag, profile "after", and diff the two.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pprof-demo/internal/workload"
)

const (
	leakEntrySize = 256 << 10 // 256 KiB retained per /leak request
	leakSafetyCap = 512 << 20 // demo stops retaining at 512 MiB
	boundedMax    = 10        // fixed version keeps only the last 10 buffers
	lockWork      = 20 * time.Millisecond
)

type Handler struct {
	leaky   *workload.LeakyStore
	bounded *workload.BoundedStore
	slowCtr *workload.SlowCounter
	fastCtr *workload.FastCounter
}

func New() *Handler {
	return &Handler{
		leaky:   workload.NewLeakyStore(leakSafetyCap),
		bounded: workload.NewBoundedStore(boundedMax),
		slowCtr: workload.NewSlowCounter(lockWork),
		fastCtr: workload.NewFastCounter(lockWork),
	}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	api := r.Group("/api")
	api.GET("/cpu", h.cpu)             // CPU profile
	api.GET("/leak", h.leak)           // heap profile (inuse_space)
	api.GET("/goroutine", h.goroutine) // goroutine profile
	api.GET("/lock", h.lock)           // mutex profile
	api.GET("/alloc", h.alloc)         // allocs profile (alloc_space)
}

func isFixed(c *gin.Context) bool { return c.Query("fixed") == "true" }

func mode(fixed bool) string {
	if fixed {
		return "fixed"
	}
	return "slow"
}

// GET /api/cpu?n=35&fixed=true
func (h *Handler) cpu(c *gin.Context) {
	n, _ := strconv.Atoi(c.DefaultQuery("n", "35"))
	if n < 0 || n > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n must be between 0 and 40"})
		return
	}
	fixed := isFixed(c)
	var result int
	if fixed {
		result = workload.FibFast(n)
	} else {
		result = workload.FibSlow(n)
	}
	c.JSON(http.StatusOK, gin.H{"mode": mode(fixed), "n": n, "fib": result})
}

// GET /api/leak?fixed=true
func (h *Handler) leak(c *gin.Context) {
	if isFixed(c) {
		c.JSON(http.StatusOK, gin.H{"mode": "fixed", "entries": h.bounded.Add(leakEntrySize)})
		return
	}
	entries, ok := h.leaky.Add(leakEntrySize)
	if !ok {
		c.JSON(http.StatusInsufficientStorage, gin.H{
			"mode": "slow", "entries": entries,
			"error": "demo safety cap reached; restart the app to leak again",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mode": "slow", "entries": entries})
}

// GET /api/goroutine?fixed=true
func (h *Handler) goroutine(c *gin.Context) {
	fixed := isFixed(c)
	if fixed {
		workload.SafeGoroutine(2 * time.Second)
	} else {
		workload.LeakGoroutine()
	}
	c.JSON(http.StatusOK, gin.H{"mode": mode(fixed)})
}

// GET /api/lock?fixed=true
func (h *Handler) lock(c *gin.Context) {
	if isFixed(c) {
		c.JSON(http.StatusOK, gin.H{"mode": "fixed", "counter": h.fastCtr.Inc()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mode": "slow", "counter": h.slowCtr.Inc()})
}

// GET /api/alloc?n=3000&fixed=true
func (h *Handler) alloc(c *gin.Context) {
	n, _ := strconv.Atoi(c.DefaultQuery("n", "3000"))
	if n < 0 || n > 20000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n must be between 0 and 20000"})
		return
	}
	fixed := isFixed(c)
	var s string
	if fixed {
		s = workload.BuildFast(n)
	} else {
		s = workload.BuildSlow(n)
	}
	c.JSON(http.StatusOK, gin.H{"mode": mode(fixed), "n": n, "length": len(s)})
}
