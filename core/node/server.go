// Server mode: node just handles incoming requests and send responses in async way
package node

import (
	"context"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"
)

func (n *Node) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return

		case req := <-n.inputCh:
			n.Metrics.NewRPC(false)

			resp := n.HandleRPC(req)
			if resp != nil {
				n.transport.SendAsync(resp.To, resp)
			}
		}
	}
}

func (n *Node) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
}

func (n *Node) HandleRPC(req *msg.Message) *msg.Message {
	n.RoutingTable.Add(req.FromID, req.From)

	resp := &msg.Message{
		ID:   req.ID,
		Type: req.Type,
		To:   req.From,
		From: req.To,
	}

	switch req.Type {
	case msg.PingType:
		// add nothing

	case msg.StoreType:
		body := req.Body.(*msg.StoreBody)
		n.store(body.Key, body.Value)
		resp = nil

	case msg.FindNodeType:
		body := req.Body.(*msg.FindNodeBody)
		resp.Body = msg.FindNodeResponse{
			Nearest: n.findNode(body.TargetID),
		}

	case msg.FindValueType:
		body := req.Body.(*msg.FindValueBody)
		if value, ok := n.findValue(body.TargetID); ok {
			resp.Body = msg.FindValueResponse{Value: value.(string)}
		} else {
			resp.Body = msg.FindNodeResponse{Nearest: n.findNode(body.TargetID)}
		}

	default:
		// TODO: unknown message type error
	}

	return resp
}

func (n *Node) store(key, value string) {
	n.KVStorage.Set(key, value)
}

func (n *Node) findNode(nodeID string) []rt.PeerInfo {
	return n.RoutingTable.KClosestNodes(pid.PeerID(nodeID), n.kad.K)
}

func (n *Node) findValue(id string) (any, bool) {
	if value, ok := n.KVStorage.Get(id); ok {
		return value, true
	}
	nearestContacts := n.RoutingTable.KClosestNodes(pid.PeerID(id), n.kad.K)
	return nearestContacts, false
}
