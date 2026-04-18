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
			value string
			peers []rt.PeerInfo
			found bool // true when value returned
			ok    bool // true when RPC succeeded
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
					results <- result{ok: false}
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
func (n *Node) FindKey(ctx context.Context, key string) (string, bool) {
	hKey := hashKey(key)
	if val, ok := n.KVStorage.Get(string(hKey)); ok {
		n.Metrics.NewSearch(key, 0, true)
		return val, true
	}

	value, ok, hops := n.keyLookup(ctx, string(hKey))
	n.Metrics.NewSearch(key, hops, ok)
	return value, ok
}

// Store finds the k closest nodes to hash(key) and sends STORE to each.
// STORE is fire-and-forget so responses are not awaited.
func (n *Node) Store(ctx context.Context, key, value string) {
	targetID := hashKey(key)
	candidates := n.NodeLookup(ctx, targetID, n.kad.K)
	for _, candidate := range candidates {
		n.transport.SendAsync(candidate.Addr, n.newStoreMsg(candidate.Addr, string(targetID), value))
	}
}

// Join bootstraps the node into the network by performing NodeLookup on its own ID.
func (n *Node) Join(ctx context.Context, bootstrapID pid.PeerID, bootstrapAddr addr.Addr) {
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

func (n *Node) newFindValueMsg(to addr.Addr, targetID pid.PeerID) *msg.Message {
	return &msg.Message{
		ID:     msg.MsgID(uuid.NewString()),
		Type:   msg.FindValueType,
		To:     to,
		From:   n.addr,
		FromID: n.id,
		Body:   &msg.FindValueBody{TargetID: string(targetID)},
	}
}

func (n *Node) newStoreMsg(to addr.Addr, key, value string) *msg.Message {
	return &msg.Message{
		ID:     msg.MsgID(uuid.NewString()),
		Type:   msg.StoreType,
		To:     to,
		From:   n.addr,
		FromID: n.id,
		Body:   &msg.StoreBody{Key: key, Value: value},
	}
}

func capped(peers []rt.PeerInfo, k int) []rt.PeerInfo {
	if len(peers) <= k {
		return peers
	}
	return peers[:k]
}
