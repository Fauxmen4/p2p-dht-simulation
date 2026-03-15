package network

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	msg "my-kad-dht/internal/message"
	rt "my-kad-dht/internal/table"
)

type storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
}

type Node struct {
	id pid.PeerID

	// network address
	addr addr.Addr

	// slice of kbuckets
	RoutingTable rt.RoutingTable

	// Number of nearest contacts to return for FindNode, FindValue.
	// In papers usually the same as bucketSize (also defined as k).
	k int

	// Number of async requests to send in parallel during node lookup operation
	alpha int

	// Network simulation. It stores mapping
	// All peers can be accessed through address as in real life.
	net *Network

	KVStorage storage

	// Mailbox for inbound messages of types network.Request, network.Response.
	inputCh chan msg.Message
}

func (n *Network) NewNode(nodeID pid.PeerID, store storage) *Node {
	node := &Node{
		id:           nodeID,
		addr:         addr.GenerateAddr(),
		RoutingTable: *rt.NewRoutingTable(20, nodeID),
		k:            n.config.Kademlia.K,
		alpha:        n.config.Kademlia.Alpha,
		net:          n,
		KVStorage:    store,
		inputCh:      make(chan msg.Message),
	}

	return node
}

func (n *Node) ID() pid.PeerID {
	return n.id
}

// Run make node listening for inbound messages through the channel and handle them in sync mode (one by one)
// TODO: add context.Context?
func (n *Node) Run() {
	for message := range n.inputCh {
		req, ok := message.(*msg.Request)
		if !ok {
			// TODO: invalid message data
		}
		resp := n.Handle(req)
		n.SendResponse(resp)
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
	return n.RoutingTable.KClosestNodes(pid.PeerID(nodeID), n.k)
}

func (n *Node) findValue(id string) (any, bool) {
	if value, ok := n.KVStorage.Get(id); ok {
		return value, true
	}
	nearestContacts := n.RoutingTable.KClosestNodes(pid.PeerID(id), n.k)
	return nearestContacts, false
}
