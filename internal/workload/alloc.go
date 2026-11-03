package workload

import (
	"strconv"
	"strings"
)

// BuildSlow is the PROBLEM version: every += allocates a brand new string and
// copies the old one, creating O(n^2) bytes of garbage.
// Use `-sample_index=alloc_space` to see it, and GC functions in the CPU profile.
func BuildSlow(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += strconv.Itoa(i) + ","
	}
	return s
}

// BuildFast is the FIXED version: one buffer, grown once.
func BuildFast(n int) string {
	var b strings.Builder
	b.Grow(n * 6) // rough upper bound for "<number>,"
	for i := 0; i < n; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteByte(',')
	}
	return b.String()
}
