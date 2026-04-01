package network

import (
	"my-kad-dht/config"
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	msg "my-kad-dht/internal/message"
	"my-kad-dht/internal/metrics"
	rt "my-kad-dht/internal/table"
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
	kad config.Kademlia

	// Network simulation. It stores mapping: address->node.
	// All peers can be accessed through address as in real life.
	net *Network

	// Mailbox for inbound messages of types network.Request, network.Response.
	inputCh chan msg.Message

	KVStorage storage

	Metrics *metrics.Storage
}

func (n *Network) NewNode(nodeID pid.PeerID, store storage) *Node {
	node := &Node{
		id:           nodeID,
		addr:         addr.GenerateAddr(),
		RoutingTable: *rt.NewRoutingTable(n.config.Kademlia.K, n.config.Kademlia.BitSize, nodeID),

		kad: n.config.Kademlia,

		net:       n,
		inputCh:   make(chan msg.Message),
		KVStorage: store,
		Metrics:   metrics.NewStorage(),
	}

	return node
}

func (n *Node) ID() pid.PeerID {
	return n.id
}

// Run make node listening for inbound messages through the channel and handle them in sync mode (one by one)
// ! TODO: add context.Context?
func (n *Node) Run() {
	for message := range n.inputCh {
		req, ok := message.(*msg.Request)
		if !ok {
			// TODO: invalid message data
		}
		resp := n.Handle(req)

		n.Metrics.NewRPC()

		if resp != nil {
			n.SendResponse(resp)
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
