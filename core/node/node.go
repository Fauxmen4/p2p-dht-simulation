package node

import (
	"context"
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/metrics"
	cfg "my-kad-dht/core/scenario"
	strg "my-kad-dht/core/storage"
	rt "my-kad-dht/core/table"
	"sync"
)

type storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
	Print()
}

type Node struct {
	// Node properties
	id           pid.PeerID
	addr         addr.Addr       // network address
	RoutingTable rt.RoutingTable // slice of kbuckets

	// Kademlia parameters
	kad cfg.Kademlia

	// Transport interface can send requests and wait for responses
	transport Transport

	// Mailbox for inbound messages of types network.Request.
	inputCh chan *msg.Message

	// Mailbox with mutex to handle a series of FIND_NODE/FIND_VALUE responses
	pending   map[msg.MsgID]chan *msg.Message
	pendingMu sync.Mutex

	KVStorage storage

	Metrics *metrics.Storage

	cancel context.CancelFunc
}

func NewNode(nodeSpec cfg.NodeSpec, cfg cfg.Kademlia, t Transport) *Node {
	id := pid.PeerID(nodeSpec.ID)
	node := &Node{
		id:           id,
		addr:         addr.Addr(nodeSpec.Address),
		RoutingTable: *rt.NewRoutingTable(cfg.K, cfg.BitSize, id),
		kad:          cfg,
		transport:    t,
		inputCh:      make(chan *msg.Message),
		pending:      make(map[msg.MsgID]chan *msg.Message),
		KVStorage:    strg.New(),
		Metrics:      metrics.NewStorage(),
	}

	return node
}

func (n *Node) ID() pid.PeerID {
	return n.id
}

func (n *Node) Addr() addr.Addr {
	return n.addr
}

func (n *Node) Deliver(m *msg.Message) {
	n.inputCh <- m
}

// Run make node listening for inbound messages through the channel and handle them in sync mode (one by one)
// ! TODO: add context.Context?
func (n *Node) RunV0_5() {
	for message := range n.inputCh {
		n.Metrics.NewRPC(false)
		switch body := message.Body.(type) {
		case *msg.Request:
			resp := n.Handle(m)
			if resp != nil {
				n.SendResponse(resp)
			}
		case *msg.Response:
			n.pendingMu.Lock()
			ch, ok := n.pending[m.ID]
			n.pendingMu.Unlock()
			if ok {
				ch <- m
			}
		}
	}
}

// helper methods for pending requests

func (n *Node) registerPending(id msg.MsgID) chan *msg.Response {
	ch := make(chan *msg.Response)

	n.pendingMu.Lock()
	n.pending[id] = ch
	n.pendingMu.Unlock()

	return ch
}

func (n *Node) unregisterPending(id msg.MsgID) {
	n.pendingMu.Lock()
	delete(n.pending, id)
	n.pendingMu.Unlock()
}
