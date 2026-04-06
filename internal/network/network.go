package network

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	msg "my-kad-dht/internal/message"
	cfg "my-kad-dht/internal/scenario"
)

type Network struct {
	config         cfg.Kademlia        // configuration of everything
	nodes          map[addr.Addr]*Node // map with all nodes (even bootstrap), used to address messages from one node to another
	bootstrapNodes []*Node             // nodes for joining the network
}

// Network constructor
func New(cfg cfg.Kademlia, bootstrapCfg []cfg.NodeSpec) *Network {
	net := &Network{
		config: cfg,
		nodes:  make(map[addr.Addr]*Node),
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
			n.bootstrapNodes[i].Run()
		}()
	}
}

// Join make corresponding node send FIND_NODE(selfID)
// to target bootstrap node according to provided NodeSpec.
func (n *Network) Join(joinInfo cfg.NodeSpec) {
	node := n.nodes[addr.Addr(joinInfo.Address)]

	bootstrapNodes := make([]*Node, len(joinInfo.BootstrapVia))
	for i := range bootstrapNodes {
		// bootstrap node ID
		bootID := joinInfo.BootstrapVia[i]

		// bootstrap node searching
		var bootNode *Node
		for _, node := range n.bootstrapNodes {
			if node.id == pid.PeerID(bootID) {
				bootNode = node
				break
			}
		}

		// bootstrap itself
		node.Join(bootNode.id, bootNode.addr)
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
	n.nodes[m.Receiver()].inputCh <- m
}
