package network

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	msg "my-kad-dht/internal/message"
	rt "my-kad-dht/internal/table"
	"time"

	"github.com/google/uuid"
)

// Useful bindings

func (n *Node) sendFind(targetID pid.PeerID, to addr.Addr, type_ msg.MsgType) (*msg.Response, bool) {
	req := &msg.Request{
		ID:   msg.MsgID(uuid.NewString()),
		Type: type_,

		Body: msg.Body{ID: string(targetID)},

		To:     to,
		From:   n.addr,
		FromID: n.id,
	}

	ch := n.registerPending(req.ID)
	defer n.unregisterPending(req.ID)

	n.net.SendBlocking(req)

	select {
	case resp := <-ch:
		return resp, true
	case <-time.After(3 * time.Millisecond):
		return nil, false
	}
}

// TODO: func (n *Node) sendFindValue() {}

// Operations built above RPC API

func (n *Node) NodeLookup(targetID pid.PeerID, k int) []rt.PeerInfo {
	// contains non-asked nodes which would be quired on next iteration
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.kad.Alpha)
	queried := make(map[pid.PeerID]struct{})

	reduced := make([]rt.PeerInfo, 0)
	reduced = append(reduced, waitlist...)
	set := make(map[pid.PeerID]struct{}) // used for deduplication in reduced
	for _, node := range waitlist {
		set[node.Id] = struct{}{}
	}

	for len(waitlist) != 0 {
		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			resp, ok := n.sendFind(targetID, nodeInfo.Addr, msg.FindNodeType)
			if !ok {
				continue
			}
			for _, peerInfo := range resp.Body.NearestNodes {
				if peerInfo.Id == n.id {
					continue
				}
				n.RoutingTable.Add(peerInfo.Id, peerInfo.Addr)
				if _, ok := set[peerInfo.Id]; !ok {
					reduced = append(reduced, peerInfo)
					set[peerInfo.Id] = struct{}{}
				}
			}
		}

		reduced = rt.SortClosestPeers(reduced, pid.ConvertPeerID(targetID))
		waitlist = make([]rt.PeerInfo, 0, n.kad.Alpha)
		for _, nodeInfo := range reduced {
			if _, ok := queried[nodeInfo.Id]; !ok {
				waitlist = append(waitlist, nodeInfo)
			}
			if len(waitlist) == n.kad.Alpha {
				break
			}
		}
	}

	if len(reduced) < k {
		return reduced
	}
	return reduced[:k]
}

func (n *Node) Join(id pid.PeerID, addr addr.Addr) {
	// add bootstrap node to routing table
	n.RoutingTable.Add(id, addr)

	// During lookup kbuckets are filled with intermediate results.
	// TODO: maybe we could also add final result to rt
	_ = n.NodeLookup(id, n.kad.K)
}

const (
	RandStrLength = 8
)

func (n *Node) Store(key, value string) {
	targetID := hashKey(key)
	candidates := n.NodeLookup(targetID, n.kad.K)
	for _, candidate := range candidates {
		n.sendStore(string(targetID), value, candidate.Addr)
	}
}

func (n *Node) sendStore(key, value string, to addr.Addr) {
	msg_ := &msg.Request{
		ID:   msg.MsgID(uuid.NewString()),
		Type: msg.StoreType,

		Body: msg.Body{Key: key, InputValue: value},

		To:     to,
		From:   n.addr,
		FromID: n.id,
	}

	n.net.SendBlocking(msg_)
}

func (n *Node) FindKey(key string) (string, bool) {
	hKey := hashKey(key)
	if val, ok := n.KVStorage.Get(string(hKey)); ok {
		n.Metrics.NewSearch(key, 0, true)
		return val, ok
	}

	value, ok, hops := n.keyLookup(string(hKey))
	n.Metrics.NewSearch(key, hops, ok)
	return value, ok
}

func (n *Node) keyLookup(key string) (string, bool, int) {
	targetID := pid.PeerID(key)

	// contains non-asked nodes which would be quired on next iteration
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.kad.Alpha)
	queried := make(map[pid.PeerID]struct{})

	reduced := make([]rt.PeerInfo, 0)
	reduced = append(reduced, waitlist...)
	set := make(map[pid.PeerID]struct{}) // used for deduplication in reduced
	for _, node := range waitlist {
		set[node.Id] = struct{}{}
	}

	hops := 0
	for len(waitlist) != 0 {
		hops++
		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			resp, ok := n.sendFind(targetID, nodeInfo.Addr, msg.FindValueType)
			if !ok {
				continue
			}
			if resp.Body.OutputValue != "" {
				return resp.Body.OutputValue, true, hops
			}
			for _, peerInfo := range resp.Body.NearestNodes {
				if peerInfo.Id == n.id {
					continue
				}
				n.RoutingTable.Add(peerInfo.Id, peerInfo.Addr)
				if _, ok := set[peerInfo.Id]; !ok {
					reduced = append(reduced, peerInfo)
					set[peerInfo.Id] = struct{}{}
				}
			}
		}

		reduced = rt.SortClosestPeers(reduced, pid.ConvertPeerID(targetID))
		waitlist = make([]rt.PeerInfo, 0, n.kad.Alpha)
		for _, nodeInfo := range reduced {
			if _, ok := queried[nodeInfo.Id]; !ok {
				waitlist = append(waitlist, nodeInfo)
			}
			if len(waitlist) == n.kad.Alpha {
				break
			}
		}
	}

	// if len(reduced) < k {
	// 	return reduced
	// }
	// return reduced[:k]

	return "", false, hops
}
