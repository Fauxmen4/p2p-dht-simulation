package routingtable

import (
	"container/list"
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	"time"
)

type PeerInfo struct {
	Id    pid.PeerID
	dhtID pid.ID
	Addr  addr.Addr

	// optimization fields
	RTT   time.Duration
	Color uint8

	// TODO: add last usage time, etc.
}

func (p PeerInfo) DhtID() pid.ID {
	return p.dhtID
}

type Bucket struct {
	list *list.List
}

func NewBucket() *Bucket {
	b := new(Bucket)
	b.list = list.New()
	return b
}

func (b *Bucket) PushFront(p PeerInfo) {
	b.list.PushFront(p)
}

func (b *Bucket) PushBack(p PeerInfo) {
	b.list.PushBack(p)
}

func (b *Bucket) Get(id pid.PeerID) PeerInfo {
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerInfo).Id == id {
			return e.Value.(PeerInfo)
		}
	}
	return PeerInfo{}
}

func (b *Bucket) Len() int {
	return b.list.Len()
}

func (b *Bucket) Remove(id pid.PeerID) bool {
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerInfo).Id == id {
			b.list.Remove(e)
			return true
		}
	}
	return false
}

// MoveToBack looks for contact with specified ID and move it to the back of the list.
func (b *Bucket) MoveToBack(id pid.PeerID) {
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerInfo).Id == id {
			b.list.MoveToBack(e)
			return
		}
	}
}

// Front returns first peerInfo in bucket in case it exists.
// Otherwise, empty peer with false is returned
func (b *Bucket) Front() (PeerInfo, bool) {
	front := b.list.Front()
	if front == nil {
		return PeerInfo{}, false
	}
	return front.Value.(PeerInfo), true
}

// RemoveFront removes and returns first peer from the list.
// Otherwise, empty peer with false is returned
func (b *Bucket) RemoveFront() (PeerInfo, bool) {
	front := b.list.Front()
	if front == nil {
		return PeerInfo{}, false
	}
	b.list.Remove(front)
	return front.Value.(PeerInfo), true
}

// ForEach applies specified function to every peer in list
func (b *Bucket) ForEach(fn func(PeerInfo)) {
	for e := b.list.Front(); e != nil; e = e.Next() {
		fn(e.Value.(PeerInfo))
	}
}

// UpdateRTT sets the RTT for the peer with the given id.
// Returns true if the peer was found and updated.
func (b *Bucket) UpdateRTT(id pid.PeerID, rtt time.Duration) bool {
	for e := b.list.Front(); e != nil; e = e.Next() {
		p := e.Value.(PeerInfo)
		if p.Id == id {
			p.RTT = rtt
			e.Value = p
			return true
		}
	}
	return false
}

// AverageRTT returns the mean RTT across all peers that have a measured RTT > 0.
// Returns 0 if no peer has a measured RTT yet.
func (b *Bucket) AverageRTT() time.Duration {
	var total time.Duration
	var count int
	for e := b.list.Front(); e != nil; e = e.Next() {
		if rtt := e.Value.(PeerInfo).RTT; rtt > 0 {
			total += rtt
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}
