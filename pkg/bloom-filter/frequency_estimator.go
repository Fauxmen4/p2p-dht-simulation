// Package bloomfilter provides a FrequencyEstimator implementing a simplified
// TinyLFU policy (Einziger & Friedman, 2014) as used by the Shades caching scheme.
//
// Structure:
//   - CountingBloomFilter (CBF): depth×width uint8 counters, capped at 15.
//     Estimate(key) returns the minimum counter value across all depth rows —
//     this is the standard Count-Min Sketch min-estimate.
//   - Minimal Increment: when incrementing, only the counters sitting at the
//     current minimum are bumped. This prevents high-frequency items from
//     inflating counters shared with low-frequency ones.
//   - BloomFilter (doorkeeper): a single-hash plain bit-array Bloom filter placed
//     in front of the CBF. An item reaches the CBF only on its second access; on
//     the first it is merely noted in the doorkeeper. This eliminates the one-hit
//     wonder problem (rare items wasting counter space).
//   - Aging: after every Window increments, all CBF counters are halved and
//     the doorkeeper is cleared. This gives the estimator a sliding-window
//     view of the access distribution.
package bloomfilter

import "sync"

const (
	DefaultDepth  = 4    // number of independent hash functions / CBF rows
	DefaultWidth  = 512  // counters per row (and bits in the doorkeeper)
	DefaultWindow = 1024 // aging: halve counters after this many Increment calls
	maxCount      = 15   // counters are capped at this value (4-bit semantics)
)

// FrequencyEstimator approximates how many times each key has been accessed
// recently. It is safe for concurrent use.
type FrequencyEstimator struct {
	mu         sync.Mutex
	window     int
	total      int // increments since the last aging pass
	doorkeeper *BloomFilter
	cbf        *CountingBloomFilter
}

// NewDefault returns a FrequencyEstimator with sensible defaults.
func NewDefault() *FrequencyEstimator {
	return New(DefaultDepth, DefaultWidth, DefaultWindow)
}

// New creates a FrequencyEstimator with the given parameters.
func New(depth, width, window int) *FrequencyEstimator {
	return &FrequencyEstimator{
		window:     window,
		doorkeeper: newBloomFilter(width),
		cbf:        newCountingBloomFilter(depth, width),
	}
}

// Estimate returns the approximate number of recent accesses for key.
func (f *FrequencyEstimator) Estimate(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cbf.estimate(key)
}

// Increment records one access to key.
//
//   - First access: key is added to the doorkeeper; CBF is not touched.
//   - Subsequent accesses: CBF counters are incremented using Minimal Increment.
//
// Every Window calls aging is triggered: all CBF counters are halved and the
// doorkeeper is cleared.
func (f *FrequencyEstimator) Increment(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Update doorkeeper/CBF before aging so the key that triggers aging is
	// still counted in the current window rather than being misattributed to
	// the next one (age resets the doorkeeper).
	if !f.doorkeeper.Has(key) {
		f.doorkeeper.Set(key)
	} else {
		f.cbf.increment(key)
	}

	f.total++
	if f.total >= f.window {
		f.age()
	}
}

// age halves every CBF counter and resets the doorkeeper.
func (f *FrequencyEstimator) age() {
	f.cbf.age()
	f.doorkeeper.Reset()
	f.total = 0
}
