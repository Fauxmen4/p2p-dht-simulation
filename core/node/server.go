// Server mode: node just handles incoming requests and send responses in async way
package node

import (
	"context"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	rt "my-kad-dht/core/table"
)

func (n *Node) Run(ctx context.Context) {
	ctx, cancelRunning := context.WithCancel(ctx)
	n.cancel = cancelRunning
	defer cancelRunning()

	for {
		select {
		case <-ctx.Done():
			return

		case m := <-n.inputCh:
			if m.IsResponse {
				n.pendingMu.Lock()
				ch, ok := n.pending[m.ID]
				n.pendingMu.Unlock()
				if ok {
					select {
					case ch <- m: // deliver to waiting operation
					default: // operation already cancelled - drop silently
					}
				}

			} else { // just a single message that should be handled with API
				n.Metrics.NewRPC(incoming)

				n.RoutingTable.MoveToBack(m.FromID)
				n.addContact(m.FromID, m.From)
				
				resp := n.HandleRPC(m)
				n.transport.SendAsync(resp)
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
	resp := &msg.Message{
		ID:         req.ID,
		Type:       req.Type,
		To:         req.From,
		From:       req.To,
		IsResponse: true,
	}

	switch req.Type {
	case msg.PingType:
		resp.Success = true

	case msg.StoreType:
		body := req.Body.(*msg.StoreBody)
		n.store(body.Key, body.Value)
		resp.Success = true

	case msg.FindNodeType:
		body := req.Body.(*msg.FindNodeBody)
		resp.Body = msg.FindNodeResponse{
			Nearest: n.findNode(body.TargetID),
		}

	case msg.FindValueType:
		body := req.Body.(*msg.FindValueBody)
		value, found := n.findValue(body.TargetID)
		if !found && n.shadeCache != nil {
			value, found = n.shadeCache.Get(body.TargetID)
		}

		fvResp := msg.FindValueResponse{}
		if found {
			fvResp.Value = value
		} else {
			fvResp.Nearest = n.findNode(body.TargetID)
		}

		if n.shadeCache != nil {
			fvResp.IsNeeded = n.shadeCache.Seen(body.TargetID)
			fvResp.ColorNodes = n.palette.GetNodesByBitmask(n.palette.Bitmask()&^body.Bitmap, 1)
		}
		resp.Body = fvResp

	case msg.StoreCacheType:
		body := req.Body.(*msg.StoreCacheBody)
		if n.shadeCache != nil {
			n.shadeCache.Set(body.Key, body.Value)
		}
		resp.Success = true

	default:
		// TODO: unknown message type error
	}

	return resp
}

func (n *Node) store(key, value string) {
	n.KVStorage.Set(key, value)
}

func (n *Node) findNode(nodeID string) []rt.PeerInfo {
	return n.RoutingTable.KClosestNodes(pid.PeerID(nodeID), n.kad.Beta)
}

func (n *Node) findValue(id string) (string, bool) {
	return n.KVStorage.Get(id)
}
