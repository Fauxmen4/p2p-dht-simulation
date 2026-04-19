package network

import (
	"context"
	"my-kad-dht/core/addr"
	"my-kad-dht/core/node"
	cfg "my-kad-dht/core/config"
)

type Network struct {
	config         cfg.Kademlia             // configuration of everything
	nodes          map[addr.Addr]*node.Node // map with all nodes (even bootstrap), used to address messages from one node to another
	bootstrapNodes []*node.Node             // nodes for joining the network
}

// Network constructor
func New(cfg cfg.Kademlia, bootstrapCfg []cfg.NodeSpec) *Network {
	net := &Network{
		config: cfg,
		nodes:  make(map[addr.Addr]*node.Node),
	}

	// bootstrap nodes
	nodes := net.CreateNNodes(bootstrapCfg, cfg)
	net.bootstrapNodes = nodes

	return net
}

// StartNetwork runs all bootstrap nodes in separate goroutines
func (n *Network) StartNetwork() {
	for i := range n.bootstrapNodes {
		go func() {
			n.bootstrapNodes[i].Run(context.Background())
		}()
	}
}
