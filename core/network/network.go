package network

import (
	"context"
	"math/rand/v2"
	"my-kad-dht/core/addr"
	cfg "my-kad-dht/core/config"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/node"
	"sync"
	"time"
)

type Network struct {
	config cfg.Kademlia // configuration of everything

	mu      sync.RWMutex
	nodes   map[addr.Addr]*node.Node // map with all nodes (even bootstrap), used to address messages from one node to another
	latency map[addr.Addr]time.Duration

	bootstrapNodes []*node.Node // nodes for joining the network
}

// Network constructor
func New(cfg cfg.Config, bootstrapCfg []cfg.NodeSpec) *Network {
	net := &Network{
		config:  cfg.Kademlia,
		mu:      sync.RWMutex{},
		nodes:   make(map[addr.Addr]*node.Node),
		latency: make(map[addr.Addr]time.Duration),
	}

	// bootstrap nodes
	nodes := net.CreateNNodes(bootstrapCfg, cfg.Kademlia)
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
	base := net.latency[to]
	net.mu.RUnlock()
	if !ok {
		return
	}
	go func() {
		if base > 0 {
			jitter := time.Duration(float64(base) * rand.Float64())
			delay := base + jitter
			delay = max(delay, 0)
			time.Sleep(delay)
		}
		n.InputCh() <- m
	}()
}
