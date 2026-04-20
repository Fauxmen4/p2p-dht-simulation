package routingtable

import (
	"fmt"
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	"sync"
)

type RoutingTable struct {
	mu sync.RWMutex

	// ID of the peer who is owner of this routing table
	selfID    pid.PeerID
	selfDhtId pid.ID
	// kbuckets
	bitSize    int // number of bits in ID, also means number of buckets
	buckets    []*Bucket
	bucketSize int
}

func NewRoutingTable(bucketSize int, bitSize int, selfID pid.PeerID) *RoutingTable {
	buckets := make([]*Bucket, bitSize)
	for i := range buckets {
		buckets[i] = NewBucket()
	}

	rt := &RoutingTable{
		selfID:    selfID,
		selfDhtId: pid.ConvertPeerID(selfID),

		bitSize:    bitSize,
		buckets:    buckets,
		bucketSize: bucketSize,
	}

	return rt
}

// LeastRecentlySeen returns earliest seen peer or (nil, false)
func (rt *RoutingTable) LeastRecentlySeen(p pid.PeerID) (PeerInfo, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	b := rt.buckets[rt.bucketIndex(p)]
	if b.Len() < rt.bucketSize {
		return PeerInfo{}, false
	}
	front := b.list.Front()
	if front == nil {
		return PeerInfo{}, false
	}
	return front.Value.(PeerInfo), true
}

func (rt *RoutingTable) MoveToBack(p pid.PeerID) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	b := rt.buckets[rt.bucketIndex(p)]
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerInfo).Id == p {
			b.list.MoveToBack(e)
			return
		}
	}
}

// Add adds new node contact to corresponding rounting table.
func (rt *RoutingTable) Add(p pid.PeerID, addr addr.Addr) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.addLocked(p, addr)
}

func (rt *RoutingTable) addLocked(p pid.PeerID, addr addr.Addr) bool {
	index := rt.bucketIndex(p)
	bucket := rt.buckets[index]

	if peerInfo := bucket.Get(p); peerInfo.Id != "" {
		return false
	}

	if bucket.Len() < rt.bucketSize {
		bucket.PushBack(PeerInfo{
			Id:    p,
			dhtID: pid.ConvertPeerID(p),
			Addr:  addr,
		})
		return true
	}

	return false
}

// KClosestNodes returns k nodes with IDs closest to target id
func (rt *RoutingTable) KClosestNodes(target pid.PeerID, k int) []PeerInfo {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	targetID := pid.ConvertPeerID(target)
	cpl := pid.CommonPrefixLen(targetID, rt.selfDhtId)
	//? useless check
	if cpl >= len(rt.buckets) {
		cpl = len(rt.buckets) - 1
	}

	pds := peerDistanceSorter{
		peers:  make([]peerDistance, 0, k+rt.bucketSize),
		target: targetID,
	}

	pds.appendPeersFromList(rt.buckets[cpl].list)

	// If not enougn, add peers from all buckets to the right.
	// All buckets to the right share exactly cpl bits
	if pds.Len() < k {
		for i := cpl + 1; i < len(rt.buckets); i++ {
			pds.appendPeersFromList(rt.buckets[i].list)
		}
	}

	// If still not enough, add buckets from the left with fewer common bits
	for i := cpl - 1; i >= 0 && pds.Len() < k; i-- {
		pds.appendPeersFromList(rt.buckets[i].list)
	}

	pds.sort()
	if k < pds.Len() {
		pds.peers = pds.peers[:k]
	}

	out := make([]PeerInfo, 0, pds.Len())
	for _, p := range pds.peers {
		out = append(out, p.p)
	}
	return out
}

func (rt *RoutingTable) Print() {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	fmt.Printf("Routing table of NodeID: %s with bucketSize: %d\n", rt.selfID, rt.bucketSize)
	fmt.Printf("Total buckets: %d\n", len(rt.buckets))
	for i, b := range rt.buckets {
		if b.Len() == 0 {
			continue
		}
		fmt.Printf("Bucket: %d. Length = %d\n", i, b.Len())
		for e := b.list.Front(); e != nil; e = e.Next() {
			peerInfo := e.Value.(PeerInfo)
			fmt.Printf("- %s (%s)\n", peerInfo.Id, peerInfo.Addr)
		}
	}
}

func (rt *RoutingTable) bucketIndex(p pid.PeerID) int {
	cpl := pid.CommonPrefixLen(
		rt.selfDhtId,
		pid.ConvertPeerID(p),
	)
	bucketID := cpl
	if bucketID >= len(rt.buckets) {
		bucketID = len(rt.buckets) - 1
	}
	return bucketID
}

// ReturnAllIds returns a list of ids from every node's kbucket
func (rt *RoutingTable) ReturnAllIds() []pid.PeerID {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	ids := make([]pid.PeerID, 0)
	for _, b := range rt.buckets {
		for e := b.list.Front(); e != nil; e = e.Next() {
			ids = append(ids, e.Value.(PeerInfo).Id)
		}
	}

	return ids
}

func (rt *RoutingTable) Remove(id pid.PeerID) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	idx := rt.bucketIndex(id)
	rt.buckets[idx].Remove(id)
}

func (rt *RoutingTable) BucketIndex(p pid.PeerID) int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.bucketIndex(p)
}

// ReplaceIfDead atomically checks LRS, and if it's the same peer, replaces it with newP.
// Returns true if replacement happened.
func (rt *RoutingTable) ReplaceIfDead(lrsID pid.PeerID, newP pid.PeerID, newAddr addr.Addr) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	idx := rt.bucketIndex(newP)
	b := rt.buckets[idx]
	front := b.list.Front()
	if front == nil {
		return false
	}
	if front.Value.(PeerInfo).Id != lrsID {
		return false // кто-то уже заменил до нас
	}
	b.list.Remove(front)
	return rt.addLocked(newP, newAddr)
}
