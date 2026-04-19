package network

import (
	"context"
	"my-kad-dht/core/addr"
	cfg "my-kad-dht/core/config"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/node"
	"sync"
)

type Network struct {
	config cfg.Kademlia // configuration of everything

	mu    sync.RWMutex
	nodes map[addr.Addr]*node.Node // map with all nodes (even bootstrap), used to address messages from one node to another

	bootstrapNodes []*node.Node // nodes for joining the network
}

// Network constructor
func New(cfg cfg.Kademlia, bootstrapCfg []cfg.NodeSpec) *Network {
	net := &Network{
		config: cfg,
		mu:     sync.RWMutex{},
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
		go func(idx int) {
			n.bootstrapNodes[idx].Run(context.Background())
		}(i)
	}
}

// Fire-and-Forget
func (net *Network) SendAsync(to addr.Addr, m *msg.Message) {
	net.mu.RLock()
	n, ok := net.nodes[to]
	net.mu.RUnlock()
	if !ok {
		return
	}
	go func() { n.InputCh() <- m }()
}
