package bloomfilter

import (
	"encoding/binary"
	"hash/fnv"
)

// CountingBloomFilter is a depth×width array of uint8 counters implementing a
// Count-Min Sketch with the Minimal Increment optimisation: only the rows whose
// counter equals the current minimum are bumped on each increment.
type CountingBloomFilter struct {
	depth    int
	width    int
	counters [][]uint8 // depth × width; values in [0, maxCount]
}

func newCountingBloomFilter(depth, width int) *CountingBloomFilter {
	counters := make([][]uint8, depth)
	for i := range counters {
		counters[i] = make([]uint8, width)
	}
	return &CountingBloomFilter{depth: depth, width: width, counters: counters}
}

// estimate returns the Count-Min minimum across all rows for key.
func (c *CountingBloomFilter) estimate(key string) int {
	min := maxCount + 1
	for i := range c.depth {
		if v := int(c.counters[i][c.idx(key, i)]); v < min {
			min = v
		}
	}
	if min > maxCount {
		return 0
	}
	return min
}

// increment bumps only the rows sitting at the current minimum (Minimal Increment).
func (c *CountingBloomFilter) increment(key string) {
	minVal := uint8(maxCount + 1)
	for i := range c.depth {
		if v := c.counters[i][c.idx(key, i)]; v < minVal {
			minVal = v
		}
	}
	for i := range c.depth {
		idx := c.idx(key, i)
		if c.counters[i][idx] == minVal && minVal < maxCount {
			c.counters[i][idx]++
		}
	}
}

// age halves every counter (aging pass).
func (c *CountingBloomFilter) age() {
	for i := range c.counters {
		for j := range c.counters[i] {
			c.counters[i][j] >>= 1
		}
	}
}

// idx maps key to a counter index in row i using a seeded FNV-64a hash.
// Each row uses a different seed (Fibonacci multiplier) for independence.
func (c *CountingBloomFilter) idx(key string, i int) int {
	h := fnv.New64a()
	var seed [8]byte
	binary.LittleEndian.PutUint64(seed[:], uint64(i)*0x9e3779b97f4a7c15)
	h.Write(seed[:])
	h.Write([]byte(key))
	return int(h.Sum64() % uint64(c.width))
}
