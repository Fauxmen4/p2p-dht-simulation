package cache

import (
	bf "my-kad-dht/pkg/bloom-filter"
	"sync"
)

// ShadeCache is a fixed-capacity cache with TinyLFU admission control and
// LazyEvict eviction. Items are only admitted when their estimated frequency
// exceeds that of the current eviction candidate, preventing cache pollution
// by one-hit wonders. The circular eviction hand advances on every Set call.
type ShadeCache struct {
	mu   sync.Mutex
	cap  int
	data map[string]string
	keys []string // circular eviction buffer; grows until len == cap
	hand int      // index of the next eviction candidate
	freq *bf.FrequencyEstimator
}

func NewShadeCache(capacity int) *ShadeCache {
	window := capacity * 10
	if window < bf.DefaultWindow {
		window = bf.DefaultWindow
	}
	return &ShadeCache{
		cap:  capacity,
		data: make(map[string]string, capacity),
		keys: make([]string, 0, capacity),
		freq: bf.New(bf.DefaultDepth, bf.DefaultWidth, window),
	}
}

// Seen records one lookup for key and returns true if this node would benefit
// from caching it (IsNeeded signal). Call this on every incoming FIND_VALUE
// request before checking the local store or cache.
func (c *ShadeCache) Seen(key string) bool {
	c.freq.Increment(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wouldAdmit(key)
}

// Get returns the cached value for key, if present.
func (c *ShadeCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[key]
	return v, ok
}

// Set tries to store key→value using TinyLFU admission + LazyEvict eviction.
// Returns true if the value was admitted and stored.
func (c *ShadeCache) Set(key, value string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.data[key]; ok {
		c.data[key] = value
		return true
	}

	if len(c.keys) < c.cap {
		c.data[key] = value
		c.keys = append(c.keys, key)
		return true
	}

	// Cache is full: admit only if the new key is more popular than the victim.
	victim := c.keys[c.hand]
	if c.freq.Estimate(key) <= c.freq.Estimate(victim) {
		c.advanceHand()
		return false
	}

	delete(c.data, victim)
	c.data[key] = value
	c.keys[c.hand] = key
	c.advanceHand()
	return true
}

func (c *ShadeCache) wouldAdmit(key string) bool {
	if len(c.keys) < c.cap {
		return true
	}
	victim := c.keys[c.hand]
	return c.freq.Estimate(key) > c.freq.Estimate(victim)
}

func (c *ShadeCache) advanceHand() {
	c.hand = (c.hand + 1) % len(c.keys)
}
