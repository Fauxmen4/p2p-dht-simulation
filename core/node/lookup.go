package node

import (
	"context"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"
)

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
			peers    []rt.PeerInfo
			deadPeer pid.PeerID
			ok       bool
		}
		results := make(chan result, len(waitlist))

		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			go func(ni rt.PeerInfo) {
				resp, err := n.sendRPC(ctx, ni.Addr, n.newFindNodeMsg(ni.Addr, targetID))
				if err != nil {
					results <- result{deadPeer: ni.Id, ok: false}
					return
				}
				body := resp.Body.(msg.FindNodeResponse)
				results <- result{peers: body.Nearest, ok: true}
			}(nodeInfo)
		}

		for range len(waitlist) {
			r := <-results
			if !r.ok {
				n.RoutingTable.Remove(r.deadPeer)
				continue
			}
			for _, peer := range r.peers {
				// TODO: should I add all returned peers every FIND_smth RPC?
				if peer.Id == n.id {
					continue
				}
				n.RoutingTable.Add(peer.Id, peer.Addr) //! Should i replace for new addContact?
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

// keyLookup performs iterative FIND_VALUE lookup. Returns the value, success flag,
// and number of rounds. Uses a per-round context to cancel remaining RPCs as soon
// as any node returns the value.
func (n *Node) keyLookup(ctx context.Context, key string) (string, bool, int) {
	targetID := pid.PeerID(key)
	waitlist := n.RoutingTable.KClosestNodes(targetID, n.kad.Alpha)
	queried := make(map[pid.PeerID]struct{})

	reduced := make([]rt.PeerInfo, 0, len(waitlist))
	seen := make(map[pid.PeerID]struct{})
	for _, peer := range waitlist {
		reduced = append(reduced, peer)
		seen[peer.Id] = struct{}{}
	}

	hops := 0
	for len(waitlist) != 0 {
		select {
		case <-ctx.Done():
			return "", false, hops
		default:
		}

		hops++

		type result struct {
			value    string
			peers    []rt.PeerInfo
			deadPeer pid.PeerID
			found    bool // true when value returned
			ok       bool // true when RPC succeeded
		}
		results := make(chan result, len(waitlist))

		// Per-round context: cancelled as soon as a value is found so remaining
		// in-flight RPCs are dropped via the dispatcher's default case.
		roundCtx, cancelRound := context.WithCancel(ctx)

		for _, nodeInfo := range waitlist {
			queried[nodeInfo.Id] = struct{}{}
			go func(ni rt.PeerInfo) {
				resp, err := n.sendRPC(roundCtx, ni.Addr, n.newFindValueMsg(ni.Addr, pid.PeerID(key)))
				if err != nil {
					results <- result{deadPeer: ni.Id, ok: false}
					return
				}
				switch body := resp.Body.(type) {
				case msg.FindValueResponse:
					results <- result{value: body.Value, found: true, ok: true}
				case msg.FindNodeResponse:
					results <- result{peers: body.Nearest, ok: true}
				default:
					results <- result{ok: false}
				}
			}(nodeInfo)
		}

		total := len(waitlist)
		var foundValue string
		var valueFound bool

		for range total {
			r := <-results // always drain to avoid goroutine leaks
			if !r.ok {
				n.RoutingTable.Remove(r.deadPeer)
				continue
			}
			if r.found && !valueFound {
				foundValue = r.value
				valueFound = true
				cancelRound() // cancel remaining in-flight RPCs
				continue
			}
			if !valueFound {
				for _, peer := range r.peers {
					// TODO: should I add all returned peers every FIND_smth RPC?
					if peer.Id == n.id {
						continue
					}
					n.addContact(peer.Id, peer.Addr)
					if _, already := seen[peer.Id]; !already {
						reduced = append(reduced, peer)
						seen[peer.Id] = struct{}{}
					}
				}
			}
		}
		cancelRound()

		if valueFound {
			return foundValue, true, hops
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

	return "", false, hops
}

// FindKey looks up the value for key. Checks local storage first, then
// performs an iterative keyLookup across the network.
func (n *Node) ValueLookup(ctx context.Context, key string) (string, bool) {
	hKey := hashKey(key)
	if val, ok := n.KVStorage.Get(string(hKey)); ok {
		n.Metrics.NewSearch(key, 0, true)
		return val, true
	}

	value, ok, hops := n.keyLookup(ctx, string(hKey))
	n.Metrics.NewSearch(key, hops, ok)
	return value, ok
}
