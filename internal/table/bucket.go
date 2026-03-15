package routingtable

import (
	"container/list"
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
)

type PeerInfo struct {
	Id    pid.PeerID
	dhtID pid.ID
	Addr  addr.Addr

	// TODO: add last usage time, etc.
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
		if e.Value.(*PeerInfo).Id == id {
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

func (b *Bucket) Peers() []PeerInfo {
	peers := make([]PeerInfo, 0, b.Len())
	for e := b.list.Front(); e != nil; e = e.Next() {
		p := e.Value.(PeerInfo)
		peers = append(peers, p)
	}
	return peers
}

func (b *Bucket) PeerIDs() []pid.PeerID {
	peerIDs := make([]pid.PeerID, 0, b.Len())
	for e := b.list.Front(); e != nil; e = e.Next() {
		p := e.Value.(PeerInfo)
		peerIDs = append(peerIDs, p.Id)
	}
	return peerIDs
}
