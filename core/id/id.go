package pid

import (
	"fmt"
	"math/bits"
)

type ID []byte

func XOR(a, b []byte) []byte {
	if len(a) != len(b) {

		panic(fmt.Sprintf(
			"failed to calculate xor byte slices should be of equal size: %d != %d",
			len(a), len(b),
		))
	}
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func zeroPrefixLen(id []byte) int {
	for i, b := range id {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(id) * 8
}

func CommonPrefixLen(a, b ID) int {
	return zeroPrefixLen(XOR(a, b))
}

func Less(id1, id2 ID) bool {
	equalBytes := 0
	for i, b1 := range id1 {
		if b1 > id2[i] {
			return false
		}
		if b1 == id2[i] {
			equalBytes++
		}
	}
	return equalBytes != len(id1)
}

type PeerID string

func ConvertPeerID(id PeerID) ID {
	return ID([]byte(id))
}
