package node

import (
	"context"
	"my-kad-dht/core/addr"
	cfg "my-kad-dht/core/config"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/metrics"
	rt "my-kad-dht/core/table"
	"my-kad-dht/pkg/rtt"
	strg "my-kad-dht/pkg/storage"
	"sync"
	"time"
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
	SendAsync(m *msg.Message)
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

	Coord rtt.Coord // node coordinates in 2D + height model for delay imitating

	KVStorage storage
	Metrics   *metrics.Storage

	palette    *rt.Palette    // Shades: color-keyed routing table (nil if Shades disabled)
	shadeCache *ShadeCache    // Shades: TinyLFU-admitted cache (nil if Shades disabled)

	cancel context.CancelFunc
}

func NewNode(id pid.PeerID, addr addr.Addr, cfg cfg.Kademlia, t Transport) *Node {
	node := &Node{
		id:           id,
		addr:         addr,
		RoutingTable: rt.NewRoutingTable(id, cfg),
		kad:          cfg,
		transport:    t,
		inputCh:      make(chan *msg.Message),
		pending:      make(map[msg.MsgID]chan *msg.Message),
		KVStorage:    strg.New(),
		Metrics:      metrics.NewStorage(),
	}

	if cfg.Shades.Colors > 0 {
		node.palette = rt.NewPalette(id, uint8(cfg.Shades.Colors))
		node.shadeCache = NewShadeCache(cfg.Shades.CacheSize)
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

func (n *Node) sendRPC(ctx context.Context, to addr.Addr, m *msg.Message) (*msg.Message, error) {
	n.Metrics.NewRPC(outgoing)

	respCh := make(chan *msg.Message, 1) // buffered — dispatcher never blocks

	n.pendingMu.Lock()
	n.pending[m.ID] = respCh
	n.pendingMu.Unlock()

	defer func() {
		n.pendingMu.Lock()
		delete(n.pending, m.ID)
		n.pendingMu.Unlock()
	}()

	n.transport.SendAsync(m)

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err() // late responses will hit `default` in dispatcher
	}
}

func (n *Node) addContact(id pid.PeerID, address addr.Addr) {
	if n.RoutingTable.Add(id, address) {
		return
	}

	lrs, ok := n.RoutingTable.LeastRecentlySeen(id)
	if !ok {
		return
	}

	//! is it correct to do it async way?
	// ping earliest seen and replace it in case its dead
	go func(id pid.PeerID, address addr.Addr) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if n.Ping(ctx, lrs) {
			n.RoutingTable.MoveToBack(lrs.Id)
		} else {
			n.RoutingTable.ReplaceIfDead(lrs.Id, id, address)
		}
	}(id, address)
}
