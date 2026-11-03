// Package config reads runtime settings from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// AppAddr is the public API address (Gin).
	AppAddr string
	// PprofAddr is the INTERNAL diagnostics address. Keep it on localhost.
	PprofAddr string
	// EnablePprof turns the diagnostics server on or off.
	EnablePprof bool
	// MutexFraction: report 1 in N mutex contention events (0 = off).
	MutexFraction int
	// BlockRateNs: record blocking events lasting >= this many nanoseconds (0 = off, 1 = everything).
	BlockRateNs int
	// StatsInterval: how often to log goroutine count and heap size (0 = off).
	StatsInterval time.Duration
}

func Load() Config {
	return Config{
		AppAddr:       getenv("APP_ADDR", ":8080"),
		PprofAddr:     getenv("PPROF_ADDR", "localhost:6060"),
		EnablePprof:   getbool("ENABLE_PPROF", true),
		MutexFraction: getint("MUTEX_FRACTION", 5),
		BlockRateNs:   getint("BLOCK_RATE_NS", 1000),
		StatsInterval: getdur("STATS_INTERVAL", 10*time.Second),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getint(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}

func getbool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getdur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
