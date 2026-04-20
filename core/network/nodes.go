package network

import (
	"context"
	"time"

	cfg "my-kad-dht/core/config"
	pid "my-kad-dht/core/id"
	"my-kad-dht/core/node"
)

func (net *Network) CreateNNodes(nodesCfg []cfg.NodeSpec, kademliaCfg cfg.Kademlia) []*node.Node {
	nodes := make([]*node.Node, len(nodesCfg))
	for i := range nodes {
		curr := node.NewNode(nodesCfg[i].ID, nodesCfg[i].Addr, kademliaCfg, net)
		net.mu.Lock()
		net.nodes[curr.Addr()] = curr
		net.latency[curr.Addr()] = nodesCfg[i].Latency
		net.mu.Unlock()
		nodes[i] = curr
	}
	return nodes
}

// Join make corresponding node send FIND_NODE(selfID)
// to target bootstrap node according to provided NodeSpec.
func (n *Network) Join(joinInfo cfg.NodeSpec) {
	targetNode := n.nodes[joinInfo.Addr]

	for _, bootID := range joinInfo.BootstrapVia {
		bootNode := n.findByID(bootID)
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

// Remove makes node leave network and wipe info about it.
func (n *Network) Remove(node *node.Node) {
	node.Stop()
	n.mu.Lock()
	delete(n.nodes, node.Addr())
	n.mu.Unlock()
}

// AddAndJoin adds info about single node and runs it.
func (n *Network) AddAndJoin(spec cfg.NodeSpec, kadCfg cfg.Kademlia) *node.Node {
	newNode := n.CreateNNodes([]cfg.NodeSpec{spec}, kadCfg)[0]
	go newNode.Run(context.Background())
	time.Sleep(10 * time.Millisecond)

	n.Join(spec)

	return newNode
}
