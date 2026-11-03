# pprof-demo: learn Go profiling with a runnable project

A small Gin service with **five deliberate performance problems**, each with a **fixed version** behind `?fixed=true`.
You profile the problem, find the cause with pprof, flip to the fix, and prove it with a second profile.

```
pprof-demo/
├── go.mod
├── Makefile                      # every command you need: make help
├── README.md
├── .gitignore
├── cmd/
│   └── server/
│       └── main.go               # wiring: config -> pprof server -> Gin -> graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go             # env settings (ports, mutex/block sampling rates)
│   ├── debugserver/
│   │   └── debugserver.go        # pprof + expvar on localhost:6060, runtime stats logger
│   ├── handlers/
│   │   └── handlers.go           # Gin routes: /api/cpu /leak /goroutine /lock /alloc
│   └── workload/                 # the slow and fixed implementations (pure Go, no Gin)
│       ├── cpu.go                # FibSlow / FibFast
│       ├── memory.go             # LeakyStore / BoundedStore
│       ├── goroutine.go          # LeakGoroutine / SafeGoroutine
│       ├── contention.go         # SlowCounter (sleep under lock) / FastCounter (atomic)
│       ├── alloc.go              # BuildSlow (+=) / BuildFast (strings.Builder)
│       └── workload_test.go      # correctness tests + leak tests + benchmarks
├── scripts/
│   ├── load.sh                   # generate load for one scenario (hey, or curl fallback)
│   ├── capture.sh                # save every profile type into profiles/<timestamp>/
│   └── leak-diff.sh              # heap snapshot -> load -> snapshot -> diff
└── profiles/                     # captured profiles land here (git-ignored)
```

**Why this layout?**
- `workload/` has no HTTP code, so it can be tested and benchmarked on its own.
- `handlers/` is thin: parse the request, call `workload`, return JSON.
- `debugserver/` runs pprof on **its own port bound to localhost**. Gin does not use `http.DefaultServeMux`, so a plain `import _ "net/http/pprof"` would not be reachable through the Gin router. A separate port also keeps profiling endpoints off the public API.

---

## 1. Run it

Requirements: Go 1.22+, `curl`. Optional: `hey` (`go install github.com/rakyll/hey@latest`) and Graphviz (`brew install graphviz` / `apt install graphviz`) for the graph view.

```bash
cd pprof-demo
go mod tidy
make test        # unit tests + leak tests
make race        # same, with the race detector
make run         # API on :8080, pprof on localhost:6060
```

Check it is alive:

```bash
curl localhost:8080/health
curl "localhost:8080/api/cpu?n=20"
curl localhost:6060/debug/pprof/            # the pprof index page
```

All other `make` targets are listed by `make help`. Scripts are called through `bash`, so no `chmod +x` is needed.

| Env var | Default | Meaning |
|---|---|---|
| `APP_ADDR` | `:8080` | API address |
| `PPROF_ADDR` | `localhost:6060` | diagnostics address (keep it on localhost) |
| `ENABLE_PPROF` | `true` | turn the diagnostics server off entirely |
| `MUTEX_FRACTION` | `5` | sample 1 in N lock-contention events (0 = off) |
| `BLOCK_RATE_NS` | `1000` | record waits of at least this many ns (0 = off) |
| `STATS_INTERVAL` | `10s` | log goroutines / heap every interval |

---

## 2. The five scenarios

| Endpoint | Problem | Profile that finds it | Fix (`?fixed=true`) |
|---|---|---|---|
| `/api/cpu` | exponential recursive Fibonacci | **CPU** | iterative O(n) |
| `/api/leak` | store keeps every 256 KiB buffer forever | **Heap** `inuse_space` | ring buffer of the last 10 |
| `/api/goroutine` | goroutine blocked on a channel forever | **Goroutine** | `select` with a context timeout |
| `/api/lock` | slow work done while holding a mutex | **Mutex** | work outside the lock, atomic counter |
| `/api/alloc` | `s += ...` in a loop | **Allocs** `alloc_space` | `strings.Builder` |

Safety: `/api/leak` stops retaining at 512 MiB and returns `507`, so the demo cannot take down your machine. Restart the app to leak again.

---

## 3. Labs: do these in order

Each lab uses two terminals: **T1** runs `make run`, **T2** runs the commands below.

### Lab 1: CPU hotspot

```bash
# T2
make load-cpu &                 # sustained load
make prof-cpu                   # 30s CPU profile, opens http://localhost:8081
```

In the browser: **VIEW → Top**, then **VIEW → Flame Graph**.

Expected: `workload.FibSlow` has about **98% flat** time. That one function is the whole story.

From the terminal instead:

```bash
curl -o cpu.prof "localhost:6060/debug/pprof/profile?seconds=10"
go tool pprof -top cpu.prof
go tool pprof -list 'FibSlow' cpu.prof        # line-by-line cost
```

Fix and compare: restart load with `make load-cpu FIXED=fixed`, capture again. FibFast no longer appears in the top.

### Lab 2: Memory leak (heap diff)

```bash
make leak-diff                  # slow version
make leak-diff FIXED=fixed      # fixed version
```

The script takes a heap snapshot, sends 200 requests, takes another, and prints what grew.

Expected:

| Run | Growth | Culprit |
|---|---|---|
| slow | about **+50 MB** | `workload.(*LeakyStore).Add` |
| fixed | about **+2 MB** (capped: 10 x 256 KiB) | `workload.(*BoundedStore).Add` |

Read the **MB column**, not the percentages. In a diff the percentages are relative to a tiny baseline and look absurd.

Manual version:

```bash
curl -o before.prof "localhost:6060/debug/pprof/heap?gc=1"
make load-leak
curl -o after.prof  "localhost:6060/debug/pprof/heap?gc=1"
go tool pprof -sample_index=inuse_space -top -diff_base=before.prof after.prof
go tool pprof -http=:8081 -diff_base=before.prof after.prof     # flame graph of the growth
```

### Lab 3: Goroutine leak

```bash
make goroutines                 # note "goroutine profile: total N"
make load-goroutine             # slow version (200 requests with the curl fallback; more with hey)
make goroutines                 # total jumped up and never comes back
```

Expected output (the leak shows as one stack shared by many goroutines; the numbers below are for the 200-request curl fallback):

```
goroutine profile: total 208
200 @ 0x475d8e 0x40f185 ...
#   workload.LeakGoroutine.func1+0x18   internal/workload/goroutine.go:14
```

Read `200 @` as "200 goroutines are parked on exactly this stack".
More detail with wait times: `curl "localhost:6060/debug/pprof/goroutine?debug=2" | less`

Fix and compare:

```bash
make load-goroutine FIXED=fixed
make goroutines                 # count jumps up...
sleep 3; make goroutines        # ...then drops back after the 2s timeout
```

Watch it live in the app log too: `[stats] goroutines=...` prints every 10 s.

### Lab 4: Lock contention

```bash
make load-lock                  # slow version, 20 parallel clients
make prof-mutex                 # opens browser UI
```

How to read it: the flat time sits on `sync.(*Mutex).Unlock`; that is normal for Go's mutex profile. The code that **caused** the waiting is the caller. Find it with:

```bash
go tool pprof -peek 'SlowCounter..Inc' http://localhost:6060/debug/pprof/mutex
```

Expected: `handlers.(*Handler).lock` -> `workload.(*SlowCounter).Inc` -> `sync.(*Mutex).Unlock`, carrying essentially all of the wait.

Why it happens: `Inc` sleeps 20 ms **while holding the lock**, so 20 clients form a queue.
Fix: `make load-lock FIXED=fixed` (work happens outside the lock; the counter is an atomic).
Throughput check with `hey`: slow is capped near 50 req/s (1 / 20 ms); fixed scales with the number of clients.

### Lab 5: Allocation churn

```bash
make load-alloc
go tool pprof -sample_index=alloc_space -top -nodecount=5 http://localhost:6060/debug/pprof/allocs
```

Expected: `workload.BuildSlow` is about **99% of all allocated bytes** (GBs for a few hundred requests).

Prove the fix without the server, using benchmarks:

```bash
make bench
# BenchmarkBuildSlow   ... B/op very large, many allocs/op
# BenchmarkBuildFast   ... one or two allocs/op
```

Statistically sound before/after comparison:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
go test -bench=Build -benchmem -count=10 ./internal/workload > old.txt
# switch the code under test, then:
go test -bench=Build -benchmem -count=10 ./internal/workload > new.txt
benchstat old.txt new.txt
```

### Lab 6: Capture everything, then explore offline

```bash
make load-cpu &                 # keep some load running
make capture                    # writes profiles/<timestamp>/ with cpu, heap, allocs,
                                # goroutine(+txt), mutex, block, threadcreate, trace
```

Open any of them later:

```bash
go tool pprof -http=:8081 profiles/<ts>/cpu.prof
go tool trace profiles/<ts>/trace.out
```

Saved files are what you attach to a bug report or compare after a deploy.

---

## 4. Command cheat sheet

```bash
make help                       # all targets

# live, from the running app
make prof-cpu                   # DURATION=30 (seconds), WEB_PORT=8081
make prof-heap  prof-allocs  prof-goroutine  prof-mutex  prof-block
make goroutines                 # quick text view
make trace                      # 5s execution trace

# evidence
make capture
make leak-diff [FIXED=fixed]
make diff-heap BASE=a.prof NEW=b.prof

# code-level
make bench                      # ns/op, B/op, allocs/op
make bench-prof                 # benchmarks that also write cpu + mem profiles
make escape                     # which variables escape to the heap
make race                       # race detector
```

Inside `go tool pprof`:

```
top  top -cum  list <func>  peek <func>  tree  web
focus=<regex>  ignore=<regex>  sample_index=alloc_space  help  quit
```

| Column | Meaning |
|---|---|
| flat | cost inside this function itself |
| cum | this function plus everything it calls |

High flat means the function's own code is hot. High cum with low flat means look at what it calls.

---

## 5. Use it on your own service

1. Copy `internal/debugserver/` into your project.
2. In `main()`: `debugserver.Start("localhost:6060", 5, 1000)`. In production, raise the block rate (for example `10000`) because `1000` records many events.
3. Keep it on localhost and reach it with `kubectl port-forward pod/<name> 6060:6060`.
4. Never expose `/debug/pprof` publicly. A CPU profile request is a cheap way to burn someone's CPU, and the output reveals internals.

Do not set an `http.Server.WriteTimeout` on the diagnostics server; `?seconds=30` requests would be cut off.

---

## 6. Troubleshooting

| Symptom | Cause / fix |
|---|---|
| CPU profile has `Total samples = 0` | app was idle; run a `make load-*` while profiling |
| Mutex / block profile empty | sampling is off; keep `MUTEX_FRACTION` and `BLOCK_RATE_NS` above 0 |
| `web` / Graph view fails | Graphviz is missing; install it (Top, Flame Graph and Source views work without it) |
| `Unknown scenario` from load.sh | valid names: `cpu leak goroutine lock alloc` |
| `/api/leak` returns 507 | safety cap reached; restart the app |
| Port 8081 busy | `make prof-cpu WEB_PORT=8090` |
| `go: ... no such host` on `go mod tidy` | your network blocks `proxy.golang.org`; set `GOPROXY` to your mirror |
