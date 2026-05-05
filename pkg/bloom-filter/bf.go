package bloomfilter

import (
	"hash/fnv"
)

// BloomFilter is a plain bit-array bloom filter with a single hash function.
// Used as a doorkeeper in front of the CountingBloomFilter.
type BloomFilter struct {
	bits  []uint64
	width int // number of addressable bits
}

func newBloomFilter(width int) *BloomFilter {
	return &BloomFilter{
		bits:  make([]uint64, (width+63)/64), // upper bound to include all "width" bits
		width: width,
	}
}

func (b *BloomFilter) Has(key string) bool {
	w, bit := b.pos(key)
	return (b.bits[w]>>bit)&1 == 1
}

func (b *BloomFilter) Set(key string) {
	w, bit := b.pos(key)
	b.bits[w] |= 1 << bit
}

func (b *BloomFilter) Reset() {
	for i := range b.bits {
		b.bits[i] = 0
	}
}

// pos maps key to a single bit position using FNV-64a with a 0xff prefix so
// it is independent of the CountingBloomFilter's row hashes.
func (b *BloomFilter) pos(key string) (word int, bit uint) {
	h := fnv.New64a()
	h.Write([]byte{0xff})
	h.Write([]byte(key))
	p := h.Sum64() % uint64(b.width)
	return int(p / 64), uint(p % 64)
}
