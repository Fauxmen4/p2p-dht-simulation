// TODO: replace uint64 for []uint64 to increase possible size
package bitset

import "fmt"

type BitSet struct {
	data uint64
}

func assertSize(index uint8) {
	if index >= 64 {
		panic(fmt.Sprintf("bitset size exceeded: %d >= 64", index))
	}
}

func (b *BitSet) Set(index uint8) {
	assertSize(index)
	b.data |= 1 << index
}

func (b *BitSet) Unset(index uint8) {
	assertSize(index)
	b.data &^= 1 << index
}

func (b *BitSet) Bits() uint64 {
	return b.data
}
