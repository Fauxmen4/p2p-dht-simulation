package network

import (
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

	// Network cfgulation. It stores mapping: address->node.
	// All peers can be accessed through address as in real life.
	net *Network

	// Mailbox for inbound messages of types network.Request.
	inputCh chan msg.Message
	// Mailbox with mutex to handle a series of FIND_NODE/FIND_VALUE responses
	pending   map[msg.MsgID]chan *msg.Response
	pendingMu sync.Mutex

	KVStorage storage

	Metrics *metrics.Storage
}

func (net *Network) NewNode(nodeSpec cfg.NodeSpec, cfg cfg.Kademlia) *Node {
	id := pid.PeerID(nodeSpec.ID)
	node := &Node{
		id:           id,
		addr:         addr.Addr(nodeSpec.Address),
		RoutingTable: *rt.NewRoutingTable(cfg.K, cfg.BitSize, id),
		kad:          cfg,
		net:          net,
		inputCh:      make(chan msg.Message),
		pending:      make(map[msg.MsgID]chan *msg.Response),
		KVStorage:    strg.New(),
		Metrics:      metrics.NewStorage(),
	}
	net.nodes[node.addr] = node
	return node
}

func (n *Node) ID() pid.PeerID {
	return n.id
}

func (n *Node) Addr() addr.Addr {
	return n.addr
}

// Run make node listening for inbound messages through the channel and handle them in sync mode (one by one)
// ! TODO: add context.Context?
func (n *Node) Run() {
	for message := range n.inputCh {
		n.Metrics.NewRPC(false)
		switch m := message.(type) {
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

func (n *Node) Handle(req *msg.Request) *msg.Response {
	n.RoutingTable.Add(req.FromID, req.From)

	resp := &msg.Response{
		ID:     req.ID,
		Type:   req.Type,
		To:     req.From,
		From:   req.To,
		FromID: n.id,
	}

	switch req.Type {
	case msg.PingType:
		// TODO:

	case msg.StoreType:
		n.store(req.Body.Key, req.Body.InputValue)
		resp = nil

	case msg.FindNodeType:
		peersInfo := n.findNode(req.Body.ID)
		resp.Body.NearestNodes = peersInfo

	case msg.FindValueType:
		result, ok := n.findValue(req.Body.ID)
		if ok {
			resp.Body.OutputValue = result.(string)
		} else {
			resp.Body.NearestNodes = result.([]rt.PeerInfo)
		}

	default:
		// TODO: unknown message type error
	}

	return resp
}

func (n *Node) SendResponse(resp *msg.Response) {
	n.Metrics.NewRPC(true)
	n.net.Send(resp)
}

// Below Kademlia RPC API implementation

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
