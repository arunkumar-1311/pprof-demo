APP       := pprof-demo
PPROF_URL := http://localhost:6060/debug/pprof
DURATION  ?= 30
WEB_PORT  ?= 8081

.DEFAULT_GOAL := help
.PHONY: help run build test race vet bench bench-prof escape \
        load-cpu load-leak load-goroutine load-lock load-alloc \
        prof-cpu prof-heap prof-allocs prof-goroutine prof-mutex prof-block \
        goroutines trace capture leak-diff diff-heap clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-16s %s\n",$$1,$$2}'

# ---------- build / quality ----------
run: ## Start the app (API :8080, pprof localhost:6060)
	go run ./cmd/server

build: ## Build binary into bin/
	go build -o bin/$(APP) ./cmd/server

test: ## Unit tests
	go test ./...

race: ## Unit tests with the race detector
	go test -race ./...

vet: ## go vet
	go vet ./...

bench: ## Benchmarks with allocation stats
	go test -bench=. -benchmem ./internal/workload/

bench-prof: ## Benchmarks that also write CPU + memory profiles
	go test -bench=. -benchmem -cpuprofile=profiles/bench-cpu.prof -memprofile=profiles/bench-mem.prof -o profiles/workload.test ./internal/workload/
	@echo "open: go tool pprof -top profiles/bench-cpu.prof"

escape: ## Escape analysis: which variables move to the heap
	go build -gcflags='-m' ./internal/... 2>&1 | grep -E 'escapes to heap|moved to heap' | head -40

# ---------- load (run in a second terminal) ----------
load-cpu: ## Load: CPU scenario        (add FIXED=fixed for the fix)
	bash ./scripts/load.sh cpu $(FIXED)

load-leak: ## Load: memory leak        (200 requests, ~50 MiB)
	bash ./scripts/load.sh leak $(FIXED)

load-goroutine: ## Load: goroutine leak
	bash ./scripts/load.sh goroutine $(FIXED)

load-lock: ## Load: lock contention
	bash ./scripts/load.sh lock $(FIXED)

load-alloc: ## Load: allocation churn
	bash ./scripts/load.sh alloc $(FIXED)

# ---------- live profiling: opens the browser UI on :$(WEB_PORT) ----------
prof-cpu: ## CPU profile for $(DURATION)s -> flame graph
	go tool pprof -http=:$(WEB_PORT) "$(PPROF_URL)/profile?seconds=$(DURATION)"

prof-heap: ## Live heap (inuse_space)
	go tool pprof -http=:$(WEB_PORT) $(PPROF_URL)/heap

prof-allocs: ## Total allocations (switch SAMPLE menu to alloc_space)
	go tool pprof -http=:$(WEB_PORT) -sample_index=alloc_space $(PPROF_URL)/allocs

prof-goroutine: ## Goroutine profile
	go tool pprof -http=:$(WEB_PORT) $(PPROF_URL)/goroutine

prof-mutex: ## Mutex contention
	go tool pprof -http=:$(WEB_PORT) $(PPROF_URL)/mutex

prof-block: ## Blocking (channels, waits)
	go tool pprof -http=:$(WEB_PORT) $(PPROF_URL)/block

goroutines: ## Print goroutines grouped by stack (quick leak check)
	@curl -s "$(PPROF_URL)/goroutine?debug=1" | head -40

trace: ## 5s execution trace -> go tool trace
	curl -s -o profiles/trace.out "$(PPROF_URL)/trace?seconds=5"
	go tool trace profiles/trace.out

# ---------- evidence / comparison ----------
capture: ## Save every profile type into profiles/<timestamp>/
	bash ./scripts/capture.sh

leak-diff: ## Heap diff before/after load (FIXED=fixed to see the fix)
	bash ./scripts/leak-diff.sh $(FIXED)

diff-heap: ## Diff two saved heaps: make diff-heap BASE=a.prof NEW=b.prof
	go tool pprof -http=:$(WEB_PORT) -diff_base=$(BASE) $(NEW)

clean: ## Remove binaries and saved profiles
	rm -rf bin
	find profiles -mindepth 1 ! -name .gitkeep -delete
