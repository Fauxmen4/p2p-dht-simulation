package node

import (
	"context"

	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"

	"github.com/google/uuid"
)

/*
Operations in "client" mode:
1. NodeLookup
2. KeyLookup (NodeLookup + FIND_VALUE instead of FIND_NODE + dropping useless responses)

3. Store (NodeLookup + STORE rpcs)
4. Join (NodeLookup(selfID))
*/

func (n *Node) sendRPC(ctx context.Context, to addr.Addr, m *msg.Message) (*msg.Message, error) {
	respCh := make(chan *msg.Message, 1) // buffered — dispatcher never blocks

	n.pendingMu.Lock()
	n.pending[m.ID] = respCh
	n.pendingMu.Unlock()

	defer func() {
		n.pendingMu.Lock()
		delete(n.pending, m.ID)
		n.pendingMu.Unlock()
	}()

	n.transport.SendAsync(to, m)

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err() // late responses will hit `default` in dispatcher
	}
}

// NodeLookup performs an iterative Kademlia node lookup for targetID,
// returning the k closest peers discovered.
func (n *Node) NodeLookup(ctx context.Context, targetID pid.PeerID, k int) []rt.PeerInfo {
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.kad.Alpha)
	queried := make(map[pid.PeerID]struct{})

	reduced := make([]rt.PeerInfo, 0, len(waitlist))
	seen := make(map[pid.PeerID]struct{})
	for _, peer := range waitlist {
		reduced = append(reduced, peer)
		seen[peer.Id] = struct{}{}
	}

	for len(waitlist) != 0 {
		select {
		case <-ctx.Done():
			return capped(reduced, k)
		default:
		}

		type result struct {
			peers []rt.PeerInfo
			ok    bool
		}
		results := make(chan result, len(waitlist))

		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			go func(ni rt.PeerInfo) {
				resp, err := n.sendRPC(ctx, ni.Addr, n.newFindNodeMsg(ni.Addr, targetID))
				if err != nil {
					results <- result{ok: false}
					return
				}
				body := resp.Body.(msg.FindNodeResponse)
				results <- result{peers: body.Nearest, ok: true}
			}(nodeInfo)
		}

		for range len(waitlist) {
			r := <-results
			if !r.ok {
				continue
			}
			for _, peer := range r.peers {
				if peer.Id == n.id {
					continue
				}
				n.RoutingTable.Add(peer.Id, peer.Addr)
				if _, already := seen[peer.Id]; !already {
					reduced = append(reduced, peer)
					seen[peer.Id] = struct{}{}
				}
			}
		}

		reduced = rt.SortClosestPeers(reduced, pid.ConvertPeerID(targetID))
		waitlist = waitlist[:0]
		for _, peer := range reduced {
			if _, ok := queried[peer.Id]; !ok {
				waitlist = append(waitlist, peer)
			}
			if len(waitlist) == n.kad.Alpha {
				break
			}
		}
	}

	return capped(reduced, k)
}

// Join bootstraps the node into the network by performing NodeLookup on its own ID.
func (n *Node) Join(ctx context.Context, bootstrapID pid.PeerID, bootstrapAddr addr.Addr) {
	// add bootstrap node to routing table
	n.RoutingTable.Add(bootstrapID, bootstrapAddr)
	
	// TODO: maybe we could also add final result to rt
	n.NodeLookup(ctx, n.id, n.kad.K)
}

func (n *Node) newFindNodeMsg(to addr.Addr, targetID pid.PeerID) *msg.Message {
	return &msg.Message{
		ID:     msg.MsgID(uuid.NewString()),
		Type:   msg.FindNodeType,
		To:     to,
		From:   n.addr,
		FromID: n.id,
		Body:   &msg.FindNodeBody{TargetID: string(targetID)},
	}
}

func capped(peers []rt.PeerInfo, k int) []rt.PeerInfo {
	if len(peers) <= k {
		return peers
	}
	return peers[:k]
}
