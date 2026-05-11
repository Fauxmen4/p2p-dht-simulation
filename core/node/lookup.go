package node

import (
	"context"
	"math/big"
	"sort"
	"time"

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
		// Collect unqueried candidates in distance order, then apply KadRTT
		// RTT-based selection with the hop-count safety condition (Eq. 3).
		var unqueried []rt.PeerInfo
		for _, p := range reduced {
			if _, ok := queried[p.Id]; !ok {
				unqueried = append(unqueried, p)
			}
		}
		waitlist = n.selectKadRTT(unqueried, n.kad.Alpha, targetDhtID)
	}
	return reduced, "", false
}

// rttEstimate returns the known RTT for p, or the average RTT of its bucket as
// a fallback when p has not been directly queried yet.
func (n *Node) rttEstimate(p rt.PeerInfo) time.Duration {
	if rtt := n.RoutingTable.GetPeerRTT(p.Id); rtt > 0 {
		return rtt
	}
	return n.RoutingTable.BucketAverageRTT(p.Id)
}

// distanceLessThanDouble reports whether dRTT < 2*dKad using exact big-integer
// arithmetic over the XOR byte slices.  This is the KadRTT hop-count condition:
// selecting a peer by RTT instead of distance is safe as long as its distance to
// the target is less than twice the closest-known distance (Eq. 3 in the paper).
func distanceLessThanDouble(dRTT, dKad pid.ID) bool {
	a := new(big.Int).SetBytes(dRTT)
	b := new(big.Int).SetBytes(dKad)
	b.Lsh(b, 1) // b = 2 * dKad
	return a.Cmp(b) < 0
}

// selectKadRTT implements the KadRTT modified-FindNode candidate selection
// (Fig. 9 of the paper).  It receives unqueried candidates already sorted by
// XOR distance (closest first) and returns up to alpha peers chosen by RTT
// while keeping the hop-count upper bound equal to standard Kademlia.
//
// Selection steps:
//  1. dMin = distance of the closest candidate (index 0 after SortClosestPeers).
//  2. Re-sort candidates by ascending RTT estimate.
//  3. Take the first alpha peers where d(p, target) < 2*dMin.
//  4. Fall back to distance order if fewer than alpha were selected by RTT.
func (n *Node) selectKadRTT(unqueried []rt.PeerInfo, alpha int, targetID pid.ID) []rt.PeerInfo {
	if len(unqueried) == 0 {
		return nil
	}

	// Step 1: dMin is the distance of the peer at index 0 (closest by distance,
	// since unqueried was produced from SortClosestPeers output).
	dMin := pid.XOR(unqueried[0].DhtID(), targetID)

	// Step 2: sort a copy by ascending RTT estimate.
	byRTT := make([]rt.PeerInfo, len(unqueried))
	copy(byRTT, unqueried)
	sort.Slice(byRTT, func(i, j int) bool {
		ri := n.rttEstimate(byRTT[i])
		rj := n.rttEstimate(byRTT[j])
		if ri == 0 && rj == 0 {
			return false
		}
		if ri == 0 {
			return false // unknown RTT goes after known
		}
		if rj == 0 {
			return true
		}
		return ri < rj
	})

	// Step 3: select peers satisfying the logarithmic condition.
	selected := make([]rt.PeerInfo, 0, alpha)
	selectedSet := make(map[pid.PeerID]struct{}, alpha)
	for _, p := range byRTT {
		if len(selected) == alpha {
			break
		}
		d := pid.XOR(p.DhtID(), targetID)
		if distanceLessThanDouble(d, dMin) {
			selected = append(selected, p)
			selectedSet[p.Id] = struct{}{}
		}
	}

	// Step 4: fill remaining slots with closest-by-distance peers (standard Kademlia fallback).
	for _, p := range unqueried {
		if len(selected) == alpha {
			break
		}
		if _, already := selectedSet[p.Id]; !already {
			selected = append(selected, p)
		}
	}

	return selected
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
