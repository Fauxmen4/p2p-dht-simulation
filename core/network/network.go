package network

import (
	"context"
	"math/rand/v2"
	"my-kad-dht/core/addr"
	cfg "my-kad-dht/core/config"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/node"
	"my-kad-dht/pkg/rtt"
	"sync"
	"time"
)

type Network struct {
	config cfg.Kademlia // configuration of everything

	mu    sync.RWMutex
	nodes map[addr.Addr]*node.Node // map with all nodes (even bootstrap), used to address messages from one node to another

	dropRate float64 // probability [0, 1) of dropping any single message

	bootstrapNodes []*node.Node // nodes for joining the network
}

// Network constructor
func New(cfg cfg.Config, bootstrapCfg []cfg.NodeSpec) *Network {
	net := &Network{
		config:   cfg.Kademlia,
		mu:       sync.RWMutex{},
		nodes:    make(map[addr.Addr]*node.Node),
		dropRate: cfg.Network.DropRate,
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
	receiver, ok1 := net.nodes[m.To]
	sender, ok2 := net.nodes[m.From]
	net.mu.RUnlock()

	if !ok1 && !ok2 {
		// receiver is out
		return
	}

	go func() {
		// possible packet loss
		if net.dropRate > 0 && rand.Float64() < net.dropRate {
			return
		}

		time.Sleep(rtt.OneWayDelay(receiver.Coord, sender.Coord))

		// sending
		select {
		case receiver.InputCh() <- m:
		// to prevent leaks in case receiver is out
		case <-time.After(time.Second):
			return
		}
	}()
}
