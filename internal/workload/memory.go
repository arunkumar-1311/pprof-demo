package workload

import "sync"

// LeakyStore is the PROBLEM version: it keeps every buffer forever.
// Heap profile (inuse_space) shows Add() holding ever more memory.
//
// A safety cap stops the demo from eating the whole machine.
type LeakyStore struct {
	mu       sync.Mutex
	data     [][]byte
	bytes    int
	maxBytes int
}

func NewLeakyStore(maxBytes int) *LeakyStore {
	return &LeakyStore{maxBytes: maxBytes}
}

// Add retains a new buffer. ok is false once the safety cap is reached.
func (s *LeakyStore) Add(size int) (entries int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bytes+size > s.maxBytes {
		return len(s.data), false
	}
	s.data = append(s.data, make([]byte, size))
	s.bytes += size
	return len(s.data), true
}

// BoundedStore is the FIXED version: a ring buffer that holds at most max
// entries, so old buffers become garbage and are collected.
type BoundedStore struct {
	mu   sync.Mutex
	data [][]byte
	max  int
	next int
}

func NewBoundedStore(max int) *BoundedStore {
	if max < 1 {
		max = 1
	}
	return &BoundedStore{max: max}
}

func (s *BoundedStore) Add(size int) (entries int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf := make([]byte, size)
	if len(s.data) < s.max {
		s.data = append(s.data, buf)
	} else {
		s.data[s.next] = buf // overwrite the oldest entry
		s.next = (s.next + 1) % s.max
	}
	return len(s.data)
}
