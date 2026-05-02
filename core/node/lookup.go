package node

import (
	"context"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"
	"time"
)

type result struct {
	value    string
	peers    []rt.PeerInfo
	deadPeer pid.PeerID
	found    bool // true when value returned
	ok       bool // true when RPC succeeded
}

// NodeLookup performs an iterative Kademlia node lookup for targetID,
// returning the k closest peers discovered.
func (n *Node) NodeLookup(ctx context.Context, targetID pid.PeerID, k int) []rt.PeerInfo {
	return n.nodeLookup(ctx, targetID, k)
}

func (n *Node) nodeLookup(ctx context.Context, targetID pid.PeerID, k int) []rt.PeerInfo {
	reduced, _, _ := n.iterativeLookup(
		ctx,
		targetID, 
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

// ValueLookup looks up the value for key. Checks local storage first, then
// performs an iterative keyLookup across the network.
func (n *Node) ValueLookup(ctx context.Context, key string) (string, bool) {
	hKey := hashKey(key)
	if val, ok := n.KVStorage.Get(string(hKey)); ok {
		n.Metrics.NewSearch(key, 0, true, 0)
		return val, true
	}
	start := time.Now()
	value, ok, hops := n.keyLookup(ctx, string(hKey))
	n.Metrics.NewSearch(key, hops, ok, time.Since(start))
	return value, ok
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
	seen := make(map[pid.PeerID]struct{})
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

// sendParallel fans out rpcFn across all peers in waitlist concurrently
// and collects all results before returning.
func (n *Node) sendParallel(ctx context.Context, waitlist []rt.PeerInfo, rpcFn func(context.Context, rt.PeerInfo) result) []result {
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
