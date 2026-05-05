// Package bloomfilter provides a FrequencyEstimator implementing a simplified
// TinyLFU policy (Einziger & Friedman, 2014) as used by the Shades caching scheme.
//
// Structure:
//   - Counting Bloom Filter (CBF): depth×width uint8 counters, capped at 15.
//     Estimate(key) returns the minimum counter value across all depth rows —
//     this is the standard Count-Min Sketch min-estimate.
//   - Minimal Increment: when incrementing, only the counters sitting at the
//     current minimum are bumped. This prevents high-frequency items from
//     inflating counters shared with low-frequency ones.
//   - Doorkeeper: a single-hash plain bit-array Bloom filter placed in front
//     of the CBF. An item reaches the CBF only on its second access; on the
//     first it is merely noted in the doorkeeper. This eliminates the one-hit
//     wonder problem (rare items wasting counter space).
//   - Aging: after every Window increments, all CBF counters are halved and
//     the doorkeeper is cleared. This gives the estimator a sliding-window
//     view of the access distribution.
package bloomfilter

import (
	"encoding/binary"
	"hash/fnv"
	"sync"
)

const (
	DefaultDepth  = 4    // number of independent hash functions / CBF rows
	DefaultWidth  = 512  // counters per row (and bits in the doorkeeper)
	DefaultWindow = 1024 // aging: halve counters after this many Increment calls
	maxCount      = 15   // counters are capped at this value (4-bit semantics)
)

// FrequencyEstimator approximates how many times each key has been accessed
// recently. It is safe for concurrent use.
type FrequencyEstimator struct {
	mu     sync.Mutex
	depth  int // number of hash functions (rows in the CBF)
	width  int // number of counters per row / bits in the doorkeeper
	window int // aging interval in number of Increment calls

	counters   [][]uint8 // depth × width CBF; values are in [0, maxCount]
	doorkeeper []uint64  // flat bit array — one bit per doorkeeper slot
	total      int       // increments since the last aging pass
}

// NewDefault returns a FrequencyEstimator with sensible defaults.
func NewDefault() *FrequencyEstimator {
	return New(DefaultDepth, DefaultWidth, DefaultWindow)
}

// New creates a FrequencyEstimator with the given parameters.
// depth is the number of hash functions; width is the number of counters per
// row; window is the aging period in Increment calls.
func New(depth, width, window int) *FrequencyEstimator {
	counters := make([][]uint8, depth)
	for i := range counters {
		counters[i] = make([]uint8, width)
	}
	dkWords := (width + 63) / 64
	return &FrequencyEstimator{
		depth:      depth,
		width:      width,
		window:     window,
		counters:   counters,
		doorkeeper: make([]uint64, dkWords),
	}
}

// Estimate returns the approximate number of recent accesses for key.
// The value is the minimum counter across all CBF rows (Count-Min estimate).
func (f *FrequencyEstimator) Estimate(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.estimate(key)
}

func (f *FrequencyEstimator) estimate(key string) int {
	min := maxCount + 1
	for i := range f.depth {
		if v := int(f.counters[i][f.cbfIdx(key, i)]); v < min {
			min = v
		}
	}
	if min > maxCount {
		return 0
	}
	return min
}

// Increment records one access to key.
//
//   - First access: key is added to the doorkeeper; CBF is not touched.
//   - Subsequent accesses: CBF counters are incremented using the Minimal
//     Increment optimization — only the rows whose counter equals the current
//     minimum are updated.
//
// Every Window calls aging is triggered: all CBF counters are halved and the
// doorkeeper is cleared.
func (f *FrequencyEstimator) Increment(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Update doorkeeper/CBF before aging so the key that triggers aging is
	// still counted in the current window rather than being misattributed to
	// the next one (age() clears the doorkeeper, which would cause dkHas to
	// return false and skip the CBF update for the triggering key).
	if !f.dkHas(key) {
		f.dkSet(key)
	} else {
		// Minimal Increment: find the current minimum counter value across all rows.
		minVal := uint8(maxCount + 1)
		for i := range f.depth {
			if v := f.counters[i][f.cbfIdx(key, i)]; v < minVal {
				minVal = v
			}
		}
		// Only bump the rows that sit at the minimum (and are not yet saturated).
		for i := range f.depth {
			idx := f.cbfIdx(key, i)
			if f.counters[i][idx] == minVal && minVal < maxCount {
				f.counters[i][idx]++
			}
		}
	}

	f.total++
	if f.total >= f.window {
		f.age()
	}
}

// age halves every CBF counter and resets the doorkeeper.
// Called automatically every window Increment calls.
func (f *FrequencyEstimator) age() {
	for i := range f.counters {
		for j := range f.counters[i] {
			f.counters[i][j] >>= 1
		}
	}
	for i := range f.doorkeeper {
		f.doorkeeper[i] = 0
	}
	f.total = 0
}

// cbfIdx maps key to a counter index in row i using a seeded FNV-64a hash.
// Each row uses a different seed so the rows are effectively independent.
func (f *FrequencyEstimator) cbfIdx(key string, i int) int {
	h := fnv.New64a()
	var seed [8]byte
	// Spread rows far apart in hash space with a Fibonacci multiplier.
	binary.LittleEndian.PutUint64(seed[:], uint64(i)*0x9e3779b97f4a7c15)
	h.Write(seed[:])
	h.Write([]byte(key))
	return int(h.Sum64() % uint64(f.width))
}

// dkPos maps key to a single bit position in the doorkeeper bit array.
// Uses a different seed (0xff prefix) so it is independent of cbfIdx row 0.
func (f *FrequencyEstimator) dkPos(key string) (word int, bit uint) {
	h := fnv.New64a()
	h.Write([]byte{0xff})
	h.Write([]byte(key))
	pos := h.Sum64() % uint64(len(f.doorkeeper)*64)
	return int(pos / 64), uint(pos % 64)
}

func (f *FrequencyEstimator) dkHas(key string) bool {
	w, b := f.dkPos(key)
	return (f.doorkeeper[w]>>b)&1 == 1
}

func (f *FrequencyEstimator) dkSet(key string) {
	w, b := f.dkPos(key)
	f.doorkeeper[w] |= 1 << b
}
