package workload

import (
	"runtime"
	"testing"
	"time"
)

// ---------- correctness: fixed versions must behave like the slow ones ----------

func TestFibFastMatchesFibSlow(t *testing.T) {
	for n := 0; n <= 20; n++ {
		if got, want := FibFast(n), FibSlow(n); got != want {
			t.Fatalf("n=%d: FibFast=%d FibSlow=%d", n, got, want)
		}
	}
}

func TestBuildFastMatchesBuildSlow(t *testing.T) {
	for _, n := range []int{0, 1, 10, 500} {
		if BuildFast(n) != BuildSlow(n) {
			t.Fatalf("n=%d: outputs differ", n)
		}
	}
}

func TestBoundedStoreNeverExceedsMax(t *testing.T) {
	s := NewBoundedStore(5)
	var n int
	for i := 0; i < 100; i++ {
		n = s.Add(16)
	}
	if n != 5 {
		t.Fatalf("expected 5 entries, got %d", n)
	}
}

func TestLeakyStoreSafetyCap(t *testing.T) {
	s := NewLeakyStore(100)
	if _, ok := s.Add(60); !ok {
		t.Fatal("first add should succeed")
	}
	if _, ok := s.Add(60); ok {
		t.Fatal("second add should hit the cap")
	}
}

func TestFastCounterCounts(t *testing.T) {
	c := NewFastCounter(0)
	for i := 1; i <= 3; i++ {
		if got := c.Inc(); got != int64(i) {
			t.Fatalf("got %d want %d", got, i)
		}
	}
}

// ---------- goroutine leak: prove the leak and the fix with numbers ----------

func TestLeakGoroutineLeaks(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 50; i++ {
		LeakGoroutine()
	}
	time.Sleep(100 * time.Millisecond)
	if grew := runtime.NumGoroutine() - before; grew < 50 {
		t.Fatalf("expected >= 50 leaked goroutines, got %d", grew)
	}
}

func TestSafeGoroutineDoesNotLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 50; i++ {
		SafeGoroutine(50 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond) // let the timeouts fire
	if grew := runtime.NumGoroutine() - before; grew > 5 {
		t.Fatalf("goroutines still running after timeout: +%d", grew)
	}
}

// ---------- benchmarks: run with  go test -bench=. -benchmem ----------

func BenchmarkFibSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FibSlow(25)
	}
}

func BenchmarkFibFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FibFast(25)
	}
}

func BenchmarkBuildSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BuildSlow(3000)
	}
}

func BenchmarkBuildFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BuildFast(3000)
	}
}
