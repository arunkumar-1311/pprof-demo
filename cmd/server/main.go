package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"pprof-demo/internal/config"
	"pprof-demo/internal/debugserver"
	"pprof-demo/internal/handlers"
	"pprof-demo/internal/items"
	"pprof-demo/internal/metrics"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. Diagnostics server (pprof) on its own localhost-only port.
	var dbg *http.Server
	if cfg.EnablePprof {
		dbg = debugserver.Start(cfg.PprofAddr, cfg.MutexFraction, cfg.BlockRateNs)
	}
	go debugserver.LogRuntimeStats(ctx, cfg.StatsInterval)

	// 2. Public API (Gin).
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	m := metrics.New()
	r.Use(m.Middleware())
	m.Register(r) // GET /metrics, scraped by Prometheus (see ../prometheus)
	handlers.New().Register(r)
	items.NewHandler(items.NewStore(), m.Registerer()).Register(r) // CRUD: /api/items

	srv := &http.Server{Addr: cfg.AppAddr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("[app] listening on %s", cfg.AppAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[app] http: %v", err)
		}
	}()

	// 3. Graceful shutdown.
	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	if dbg != nil {
		_ = dbg.Shutdown(shutdownCtx)
	}
}
