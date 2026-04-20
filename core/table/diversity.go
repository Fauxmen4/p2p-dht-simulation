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
	// count contacts per slot
	slotCount := make(map[uint]int)
	for e := b.list.Front(); e != nil; e = e.Next() {
		pi := e.Value.(PeerInfo)
		slot := rt.diversitySlot(pi.Id, bucketLevel)
		slotCount[slot]++
	}

	// find LRS (front of list) with a duplicated slot
	for e := b.list.Front(); e != nil; e = e.Next() {
		pi := e.Value.(PeerInfo)
		slot := rt.diversitySlot(pi.Id, bucketLevel)
		if slotCount[slot] > 1 {
			return pi, true
		}
	}
	return PeerInfo{}, false
}

// LeastRecentlySeenDiverse returns the eviction candidate according to the
// diversity-aware policy from Salah et al. 2014:
// - If any slot is duplicated → evict LRS among those duplicates
// - Otherwise (all slots unique) → fallback to global LRS
func (rt *RoutingTable) LeastRecentlySeenDiverse(p pid.PeerID) (PeerInfo, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	bucketLevel := rt.bucketIndex(p)
	b := rt.buckets[bucketLevel]
	if b.Len() < rt.bucketSize {
		return PeerInfo{}, false
	}

	// prefer evicting a duplicate-slot contact
	if lrs, ok := rt.lrsAmongDuplicates(b, bucketLevel); ok {
		return lrs, true
	}

	// fallback: global LRS (standard Kademlia)
	front := b.list.Front()
	if front == nil {
		return PeerInfo{}, false
	}
	return front.Value.(PeerInfo), true
}
