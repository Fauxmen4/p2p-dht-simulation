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

// Store finds the k closest nodes to hash(key) and sends STORE to each
// in fire-and-forget manner.
func (n *Node) Store(ctx context.Context, key, value string) {
	targetID := hashKey(key)
	candidates := n.NodeLookup(ctx, targetID, n.kad.K)

	type result struct {
		id       pid.PeerID
		addr     addr.Addr
		deadPeer pid.PeerID
		ok       bool
	}
	results := make(chan result, len(candidates))

	for _, candidate := range candidates {
		go func(pi rt.PeerInfo) {
			m := n.newStoreMsg(candidate.Addr, string(targetID), value)
			resp, err := n.sendRPC(ctx, pi.Addr, m)
			if err != nil {
				results <- result{deadPeer: pi.Id, ok: false}
				return
			}
			results <- result{id: candidate.Id, addr: candidate.Addr, ok: resp.Success}
		}(candidate)
	}

	// cnt := len(candidates) // just statistics
	for range len(candidates) {
		r := <-results
		if !r.ok && r.deadPeer != "" {
			n.RoutingTable.Remove(r.deadPeer)
			// cnt-- // just statistics
		} else {
			n.RoutingTable.MoveToBack(r.id)
			n.addContact(r.id, r.addr)
		}
	}

	// fmt.Println("sent store to nodes:", cnt)
}

func (n *Node) Ping(ctx context.Context, pi rt.PeerInfo) bool {
	resp, err := n.sendRPC(ctx, pi.Addr, n.newPingMsg(pi.Addr))
	if err != nil {
		return false
	}
	return resp.Success
}

// Join bootstraps the node into the network by performing NodeLookup on its own ID.
func (n *Node) Join(ctx context.Context, bootstrapID pid.PeerID, bootstrapAddr addr.Addr) {
	n.RoutingTable.Add(bootstrapID, bootstrapAddr)

	// TODO: maybe we could also add final result to rt
	n.NodeLookup(ctx, n.id, n.kad.K)
}

func (n *Node) newPingMsg(to addr.Addr) *msg.Message {
	return &msg.Message{
		ID:     msg.MsgID(uuid.NewString()),
		Type:   msg.PingType,
		To:     to,
		From:   n.addr,
		FromID: n.id,
	}
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
