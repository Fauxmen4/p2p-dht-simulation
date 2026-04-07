package node

import (
	"context"
	"my-kad-dht/core/addr"
	msg "my-kad-dht/core/message"
)

/*
Operations in "client" mode:
1. NodeLookup
2. KeyLookup (NodeLookup + FIND_VALUE instead of FIND_NODE + dropping useless responses)

3. Store (NodeLookup + STORE rpcs)
4. Join (NodeLookup(targetID))

*/

func (n *Node) sendRPC(ctx context.Context, to addr.Addr, m *msg.Message) (*msg.Message, error) {
    respCh := make(chan *msg.Message, 1) // buffered - dispatcher never blocks

    n.pendingMu.Lock()
    n.pending[m.ID] =respCh
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
        return nil, ctx.Err() // late responses will hit "default" in dispatcher
    }
}


