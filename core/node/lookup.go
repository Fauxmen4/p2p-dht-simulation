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

type shadesResult struct {
	result
	isNeeded   bool
	colorNodes []rt.PeerInfo
	fromPeer   rt.PeerInfo
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
// and number of rounds. Dispatches to shadesKeyLookup when Shades is enabled.
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
					ch <- result{value: body.Value, found: body.Value != "", ok: true, peers: body.Nearest}
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
			r := <-ch
			if !r.ok {
				if r.deadPeer != "" {
					n.RoutingTable.Remove(r.deadPeer)
				}
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

// shadesKeyLookup implements the Shades(2016) FIND_VALUE algorithm:
//   - Each round sends FIND_VALUE to α closest unqueried nodes plus one side step
//     to the closest same-color node from the local palette.
//   - ColorNodes piggybacked in every response are merged into the local palette.
//   - After the value is found, STORE_CACHE is sent fire-and-forget to every node
//     that returned IsNeeded=true during the lookup.
func (n *Node) shadesKeyLookup(ctx context.Context, key string) (string, bool, int) {
	targetID := pid.PeerID(key)
	keyDhtID := pid.ConvertPeerID(targetID)
	hops := 0
	var neededNodes []rt.PeerInfo

	_, value, found := n.iterativeLookup(ctx, targetID, func(ctx context.Context, waitlist []rt.PeerInfo) ([]rt.PeerInfo, string, bool) {
		hops++

		// Build probe list: normal α waitlist nodes + side step to the closest
		// same-color node from palette (if it is not already in the waitlist).
		probes := append([]rt.PeerInfo(nil), waitlist...)
		if colorNode, ok := n.palette.ClosestToKey(keyDhtID); ok && !hasPeer(probes, colorNode.Id) {
			probes = append(probes, colorNode)
		}

		ch := make(chan shadesResult, len(probes))
		roundCtx, cancelRound := context.WithCancel(ctx)

		for _, ni := range probes {
			go func(ni rt.PeerInfo) {
				resp, err := n.sendRPC(roundCtx, ni.Addr, n.newFindValueMsg(ni.Addr, targetID))
				if err != nil {
					ch <- shadesResult{result: result{deadPeer: ni.Id}}
					return
				}
				switch body := resp.Body.(type) {
				case msg.FindValueResponse:
					r := shadesResult{
						result:     result{ok: true, peers: body.Nearest},
						isNeeded:   body.IsNeeded,
						colorNodes: body.ColorNodes,
						fromPeer:   ni,
					}
					if body.Value != "" {
						r.result.found = true
						r.result.value = body.Value
					}
					ch <- r
				case msg.FindNodeResponse:
					ch <- shadesResult{result: result{peers: body.Nearest, ok: true}}
				default:
					ch <- shadesResult{result: result{ok: false}}
				}
			}(ni)
		}

		var newPeers []rt.PeerInfo
		var foundValue string
		valueFound := false

		for range len(probes) {
			sr := <-ch
			if !sr.ok {
				if sr.deadPeer != "" {
					n.RoutingTable.Remove(sr.deadPeer)
				}
				continue
			}

			// Gossip: merge piggybacked color nodes into local palette.
			for _, cn := range sr.colorNodes {
				n.palette.Add(cn)
			}

			if sr.isNeeded {
				neededNodes = append(neededNodes, sr.fromPeer)
			}

			if sr.found && !valueFound {
				foundValue, valueFound = sr.value, true
				cancelRound()
				continue
			}

			if !valueFound {
				for _, p := range sr.peers {
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

	// Cache admission: fire-and-forget STORE_CACHE to every node that signalled IsNeeded.
	if found {
		for _, ni := range neededNodes {
			n.transport.SendAsync(n.newStoreCacheMsg(ni.Addr, key, value))
		}
	}

	return value, found, hops
}

func hasPeer(peers []rt.PeerInfo, id pid.PeerID) bool {
	for _, p := range peers {
		if p.Id == id {
			return true
		}
	}
	return false
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
