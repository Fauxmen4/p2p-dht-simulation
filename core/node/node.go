package node

import (
	"context"
	"my-kad-dht/core/addr"
	cfg "my-kad-dht/core/config"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/metrics"
	strg "my-kad-dht/core/storage"
	rt "my-kad-dht/core/table"
	"sync"
)

const (
	incoming = true
	outgoing = false
)

type storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
	Print()
}

type Transport interface {
	// SendAsync delivers a message in a fire-and-forget manner.
	SendAsync(to addr.Addr, m *msg.Message)
}

type Node struct {
	// Node properties
	id           pid.PeerID
	addr         addr.Addr        // network address
	RoutingTable *rt.RoutingTable // slice of kbuckets
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
	Metrics   *metrics.Storage

	cancel context.CancelFunc
}

func NewNode(id pid.PeerID, addr addr.Addr, cfg cfg.Kademlia, t Transport) *Node {
	node := &Node{
		id:           id,
		addr:         addr,
		RoutingTable: rt.NewRoutingTable(cfg.K, cfg.BitSize, id),
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

func (n *Node) InputCh() chan *msg.Message {
	return n.inputCh
}
