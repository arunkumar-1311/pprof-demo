package workload

// FibSlow is the PROBLEM version: exponential recursion, O(2^n).
// In a CPU profile it appears as one function with enormous flat time.
func FibSlow(n int) int {
	if n < 2 {
		return n
	}
	return FibSlow(n-1) + FibSlow(n-2)
}

// FibFast is the FIXED version: iterative, O(n).
func FibFast(n int) int {
	a, b := 0, 1
	for i := 0; i < n; i++ {
		a, b = b, a+b
	}
	return a
}
