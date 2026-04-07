package network

import (
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

	bootstrapNodes := make([]*node.Node, len(joinInfo.BootstrapVia))
	for i := range bootstrapNodes {
		// bootstrap node ID
		bootID := joinInfo.BootstrapVia[i]

		// bootstrap node searching
		var targetBootNode *node.Node
		for _, bootNode := range n.bootstrapNodes {
			if bootNode.ID() == pid.PeerID(bootID) {
				targetBootNode = bootNode
				break
			}
		}

		// bootstrap itself
		targetNode.Join(targetBootNode.ID(), targetBootNode.Addr())
	}
}

// Fire-and-Forget
func (net *Network) SendAsync(to addr.Addr, m *msg.Message) {
	go net.nodes[to].Deliver(m)
}
