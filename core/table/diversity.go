package routingtable

import (
	"math/bits"
	pid "my-kad-dht/core/id"
)

// diversitySlot extracts q bits of peer's ID starting at position 'start'.
func (rt *RoutingTable) diversitySlot(p pid.PeerID, bucketLevel int) uint {
	// q = ⌊log k⌋, number of bits to consider after prefix
    q := bits.Len(uint(rt.bucketSize)) - 1
	if q == 0 {
		return 0
	}

	dhtID := pid.ConvertPeerID(p)

	// Extract q bits starting at bit position 'bucketLevel'
	var result uint
	for i := range q {
		bitPos := bucketLevel + i
		byteIdx := bitPos / 8
		bitIdx := 7 - (bitPos % 8)
		if byteIdx < len(dhtID) {
			bit := (dhtID[byteIdx] >> bitIdx) & 1
			result = (result << 1) | uint(bit)
		}
	}

	return result
}

// lrsAmongDuplicates returns the least-recently-seen contact whose diversity
// slot is shared by at least one other contact in the bucket.
// Returns (zero, false) if all slots are unique (no duplicates exist).
func (rt *RoutingTable) lrsAmongDuplicates(b *Bucket, bucketLevel int) (PeerInfo, bool) {
	slotCount := make(map[uint]int)
	b.ForEach(func(pi PeerInfo) {
		slotCount[rt.diversitySlot(pi.Id, bucketLevel)]++
	})

	// find LRS (front of list) with a duplicated slot
	var found PeerInfo
	ok := false
	b.ForEach(func(pi PeerInfo) {
		if ok {
			return
		}
		if slotCount[rt.diversitySlot(pi.Id, bucketLevel)] > 1 {
			found, ok = pi, true
		}
	})
	return found, ok
}
