package routingtable

import (
	"container/list"
	pid "my-kad-dht/core/id"
	"sort"
)

type peerDistance struct {
	p        PeerInfo
	distance pid.ID
}

type peerDistanceSorter struct {
	peers  []peerDistance
	target pid.ID
}

// sort interface implementation
func (pds *peerDistanceSorter) Len() int {
	return len(pds.peers)
}
func (pds *peerDistanceSorter) Swap(i, j int) {
	pds.peers[i], pds.peers[j] = pds.peers[j], pds.peers[i]
}
func (pds *peerDistanceSorter) Less(i, j int) bool {
	return pid.Less(pds.peers[i].distance, pds.peers[j].distance)
}

func (pds *peerDistanceSorter) appendPeer(peerInfo PeerInfo) {
	pds.peers = append(pds.peers, peerDistance{
		p:        peerInfo,
		distance: pid.XOR(pds.target, peerInfo.dhtID),
	})
}

func (pds *peerDistanceSorter) appendPeersFromList(l *list.List) {
	for e := l.Front(); e != nil; e = e.Next() {
		pds.appendPeer(e.Value.(PeerInfo))
	}
}

func (pds *peerDistanceSorter) sort() {
	sort.Sort(pds)
}

func SortClosestPeers(peers []PeerInfo, target pid.ID) []PeerInfo {
	sorter := peerDistanceSorter{
		peers:  make([]peerDistance, 0, len(peers)),
		target: target,
	}
	for _, p := range peers {
		sorter.appendPeer(p)
	}
	sorter.sort()
	out := make([]PeerInfo, 0, sorter.Len())
	for _, p := range sorter.peers {
		out = append(out, p.p)
	}
	return out
}
