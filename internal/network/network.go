package network

import (
	"math/rand/v2"
	"time"

	"my-kad-dht/internal/addr"
	"my-kad-dht/internal/config"
	msg "my-kad-dht/internal/message"
	"my-kad-dht/internal/utils"
)

type Network struct {
	config         config.Config       // configuration of everything
	nodes          map[addr.Addr]*Node // map with all nodes, used to address messages from one node to another
	bootstrapNodes []*Node             // nodes for joining the network
}

// Network constructor
func New(cfg config.Config) *Network {
	net := &Network{
		config: cfg,
		nodes:  make(map[addr.Addr]*Node),
	}

	// bootstrap nodes
	nodes := net.CreateNNodes(cfg.Network.Bootstrap.NodesCount)
	net.bootstrapNodes = nodes
	for _, node := range nodes {
		net.nodes[node.addr] = node
	}

	return net
}

func (n *Network) Join(node *Node) {
	n.nodes[node.addr] = node

	// randomly choose bootstrap nodes
	bootstrapNodes := utils.RandomElements(
		n.bootstrapNodes,
		n.config.Network.Bootstrap.Connections_count,
	)
	for _, bootNode := range bootstrapNodes {
		node.Join(bootNode.id, bootNode.addr)
	}
}

// StartNetwork runs all bootstrap nodes in separate goroutines
func (n *Network) StartNetwork() {
	for i := range n.bootstrapNodes {
		go func() {
			n.bootstrapNodes[i].Run()
		}()
	}
}

// Send sends message from one node to other in non-blocking way
func (n *Network) Send(msg msg.Message) {
	select {
	case n.nodes[msg.Receiver()].inputCh <- msg:
	default:
	}
}

// SendBlocking sends message and blocks until reader appears.
// Applies configured drop_rate and latency_ms failure injections.
func (n *Network) SendBlocking(m msg.Message) {
	if n.config.Network.DropRate > 0 && rand.Float64() < n.config.Network.DropRate {
		return
	}
	if n.config.Network.LatencyMs > 0 {
		time.Sleep(time.Duration(n.config.Network.LatencyMs) * time.Millisecond)
	}
	n.nodes[m.Receiver()].inputCh <- m
}
