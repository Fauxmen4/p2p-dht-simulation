package node

import (
	"context"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"
)

type result struct {
	value    string
	peers    []rt.PeerInfo
	deadPeer pid.PeerID
	found    bool // true when value returned
	ok       bool // true when RPC succeeded
}

func (n *Node) nodeLookup(ctx context.Context, targetID pid.PeerID, k int) []rt.PeerInfo {
	reduced, _, _ := n.iterativeLookup(
		ctx, targetID,
		func(ctx context.Context, waitlist []rt.PeerInfo) ([]rt.PeerInfo, string, bool) {
			results := n.sendParallel(
				ctx,
				waitlist,
				func(ctx context.Context, ni rt.PeerInfo) result {
					resp, err := n.sendRPC(ctx, ni.Addr, n.newFindNodeMsg(ni.Addr, targetID))
					if err != nil {
						return result{deadPeer: ni.Id}
					}
					return result{peers: resp.Body.(msg.FindNodeResponse).Nearest, ok: true}
				},
			)

			var newPeers []rt.PeerInfo
			for _, r := range results {
				if !r.ok {
					n.RoutingTable.Remove(r.deadPeer)
					continue
				}
				for _, p := range r.peers {
					if p.Id != n.id {
						n.RoutingTable.Add(p.Id, p.Addr) //! Should i replace for new addContact?
						newPeers = append(newPeers, p)
					}
				}
			}

			return newPeers, "", false
		},
	)
	return capped(reduced, k)
}

// keyLookup performs iterative FIND_VALUE lookup. Returns the value, success flag,
// and number of rounds. Uses a per-round context to cancel remaining RPCs as soon
// as any node returns the value.
func (n *Node) keyLookup(ctx context.Context, key string) (string, bool, int) {
	targetID := pid.PeerID(key)
	hops := 0

	_, value, found := n.iterativeLookup(ctx, targetID, func(ctx context.Context, waitlist []rt.PeerInfo) ([]rt.PeerInfo, string, bool) {
		hops++
		ch := make(chan result, len(waitlist))
		roundCtx, cancelRound := context.WithCancel(ctx)
		for _, ni := range waitlist {
			go func(ni rt.PeerInfo) {
				resp, err := n.sendRPC(roundCtx, ni.Addr, n.newFindValueMsg(ni.Addr, pid.PeerID(key)))
				if err != nil {
					ch <- result{deadPeer: ni.Id}
					return
				}
				switch body := resp.Body.(type) {
				case msg.FindValueResponse:
					ch <- result{value: body.Value, found: true, ok: true}
				case msg.FindNodeResponse:
					ch <- result{peers: body.Nearest, ok: true}
				default:
					ch <- result{ok: false}
				}
			}(ni)
		}

		var newPeers []rt.PeerInfo
		var foundValue string
		
		valueFound := false

		for range len(waitlist) {
			r := <-ch // always drain to avoid goroutine leaks
			if !r.ok {
				n.RoutingTable.Remove(r.deadPeer)
				continue
			}
			if r.found && !valueFound {
				foundValue, valueFound = r.value, true
				cancelRound()
				continue
			}
			if !valueFound {
				for _, p := range r.peers {
					if p.Id != n.id {
						n.addContact(p.Id, p.Addr)
						newPeers = append(newPeers, p)
					}
				}
			}
		}
		cancelRound()
		return newPeers, foundValue, valueFound
	})

	return value, found, hops
}

// iterativeLookup runs the Kademlia iterative lookup loop.
// roundFn is called each round with the current waitlist; it must:
//   - send RPCs and remove dead peers from the routing table
//   - return newly discovered peers (unseen ones are added to the candidate set)
//   - return (_, value, true) to signal early termination with a found value
func (n *Node) iterativeLookup(
	ctx context.Context,
	targetID pid.PeerID,
	roundFn func(ctx context.Context, waitlist []rt.PeerInfo) ([]rt.PeerInfo, string, bool),
) ([]rt.PeerInfo, string, bool) {
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.kad.Alpha)
	queried := make(map[pid.PeerID]struct{})

	// deduplicator for reduce
	seen := make(map[pid.PeerID]struct{})
	// add in case contact is not in set "seen"
	reduced := make([]rt.PeerInfo, 0, len(waitlist))

	for _, p := range waitlist {
		reduced = append(reduced, p)
		seen[p.Id] = struct{}{}
	}
	targetDhtID := pid.ConvertPeerID(targetID)

	for len(waitlist) != 0 {
		if ctx.Err() != nil {
			break
		}
		for _, p := range waitlist {
			queried[p.Id] = struct{}{}
		}
		newPeers, value, found := roundFn(ctx, waitlist)
		if found {
			return reduced, value, true
		}
		for _, p := range newPeers {
			if _, already := seen[p.Id]; !already {
				reduced = append(reduced, p)
				seen[p.Id] = struct{}{}
			}
		}
		reduced = rt.SortClosestPeers(reduced, targetDhtID)
		waitlist = waitlist[:0]
		for _, p := range reduced {
			if _, ok := queried[p.Id]; !ok {
				waitlist = append(waitlist, p)
			}
			if len(waitlist) == n.kad.Alpha {
				break
			}
		}
	}
	return reduced, "", false
}

// sendParallel executes rpcFn on every node from waitlist, then fans out every response to returning channel. 
func (n *Node) sendParallel(
	ctx context.Context,
	waitlist []rt.PeerInfo, 
	rpcFn func(context.Context, rt.PeerInfo) result,
) []result {
	ch := make(chan result, len(waitlist))
	for _, ni := range waitlist {
		go func(ni rt.PeerInfo) {
			ch <- rpcFn(ctx, ni)
		}(ni)
	}
	out := make([]result, len(waitlist))
	for i := range out {
		out[i] = <-ch
	}
	return out
}
