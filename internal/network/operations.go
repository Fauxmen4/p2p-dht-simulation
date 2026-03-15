package network

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	"my-kad-dht/internal/message"
	rt "my-kad-dht/internal/table"
	"time"

	"github.com/google/uuid"
)

// Useful bindings

func (n *Node) sendFindNode(targetID pid.PeerID, to addr.Addr) {
	msg := &message.Request{
		ID:   message.MsgID(uuid.NewString()),
		Type: message.FindNodeType,

		Body: message.Body{ID: string(targetID)},

		To:     to,
		From:   n.addr,
		FromID: n.id,
	}

	n.net.SendBlocking(msg)
}

// TODO: func (n *Node) sendFindValue() {}

// Operations built above RPC API

func (n *Node) NodeLookup(targetID pid.PeerID, k int) []rt.PeerInfo {
	// contains non-asked nodes which would be quired on next iteration
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.alpha)
	queried := make(map[pid.PeerID]struct{})

	reduced := make([]rt.PeerInfo, 0)
	reduced = append(reduced, waitlist...)
	set := make(map[pid.PeerID]struct{}) // used for deduplication in reduced
	for _, node := range waitlist {
		set[node.Id] = struct{}{}
	}

	for len(waitlist) != 0 {

		// send alpha (or less) RPC FIND_NODE requests
		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			n.sendFindNode(targetID, nodeInfo.Addr)
		}

		// results from all RPCs gather to reduced,
		// than they 1. deduplicated, 2. sorted by distance to target ID, 3. choose alpha (or less) non-queried before
		outer:
		for range len(waitlist) {
			//? Here I caught deadlock, so I made reading with timeout
			//! TEMPORARY FIX
			var resp *message.Response
			select {
			case msg := (<-n.inputCh):
				resp = msg.(*message.Response)
			case <-time.After(3 * time.Millisecond):
				break outer
			}

			// add fresh data to routing table
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
		waitlist = make([]rt.PeerInfo, 0, n.alpha)
		for _, nodeInfo := range reduced {
			if _, ok := queried[nodeInfo.Id]; !ok {
				waitlist = append(waitlist, nodeInfo)
			}
			if len(waitlist) == n.alpha {
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
	_ = n.NodeLookup(id, n.k)
}
