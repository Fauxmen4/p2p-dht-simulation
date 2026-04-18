package network

import (
	"context"

	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/node"
	cfg "my-kad-dht/core/scenario"
)

func (net *Network) CreateNNodes(nodesCfg []cfg.NodeSpec, kademliaCfg cfg.Kademlia) []*node.Node {
	nodes := make([]*node.Node, len(nodesCfg))
	for i := range nodes {
		curr := node.NewNode(nodesCfg[i], kademliaCfg, net)
		net.nodes[curr.Addr()] = curr
		nodes[i] = curr
	}
	return nodes
}

// Join make corresponding node send FIND_NODE(selfID)
// to target bootstrap node according to provided NodeSpec.
func (n *Network) Join(joinInfo cfg.NodeSpec) {
	targetNode := n.nodes[addr.Addr(joinInfo.Address)]

	for _, bootID := range joinInfo.BootstrapVia {
		bootNode := n.findByID(pid.PeerID(bootID))
		if bootNode == nil {
			continue
		}
		targetNode.Join(context.Background(), bootNode.ID(), bootNode.Addr())
	}
}

// findByID looks up any node (bootstrap or regular) by PeerID.
func (n *Network) findByID(id pid.PeerID) *node.Node {
	for _, node := range n.nodes {
		if node.ID() == id {
			return node
		}
	}
	return nil
}

// Fire-and-Forget
func (net *Network) SendAsync(to addr.Addr, m *msg.Message) {
	go net.nodes[to].Deliver(m)
}
