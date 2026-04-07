package node

import (
	"context"
	"sync"

	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/metrics"
	cfg "my-kad-dht/core/scenario"
	strg "my-kad-dht/core/storage"
	rt "my-kad-dht/core/table"
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
	// For pending messages used during operation
	pendingMu sync.Mutex
	pending   map[msg.MsgID]chan *msg.Message

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

// Deliver is used in network package for async communication.
// TODO: add context, timeouts, delays, etc.
func (n *Node) Deliver(m *msg.Message) {
	n.inputCh <- m
}
