package network

import (
	"context"
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	msg "my-kad-dht/core/message"
	"my-kad-dht/core/node"
	cfg "my-kad-dht/core/scenario"
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

func (net *Network) CreateNNodes(nodesCfg []cfg.NodeSpec, kademliaCfg cfg.Kademlia) []*node.Node {
	nodes := make([]*node.Node, len(nodesCfg))
	for i := range nodes {
		curr := node.NewNode(nodesCfg[i], kademliaCfg, net)
		net.nodes[curr.Addr()] = curr
		nodes[i] = curr
	}
	return nodes
}

// StartNetwork runs all bootstrap nodes in separate goroutines
func (n *Network) StartNetwork() {
	for i := range n.bootstrapNodes {
		go func() {
			n.bootstrapNodes[i].Run(context.Background())
		}()
	}
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

func (net *Network) Call(to addr.Addr, m *msg.Message) (*msg.Message, error) {
	ch := make(chan *msg.Message, 1)
	net
}

// Fire-and-Forget
func (net *Network) SendAsync(to addr.Addr, m *msg.Message) {
	go net.nodes[to].Deliver(m)
}

//! LEGACY BULLSHIT

// Send sends message from one node to other in non-blocking way
func (n *Network) Send(msg msg.Message) {
	select {
	case n.nodes[msg.To].inputCh <- msg:
	default:
	}
}

// SendBlocking sends message and blocks until reader appears.
// Applies configured drop_rate and latency_ms failure injections.
func (n *Network) SendBlocking(m msg.Message) {
	n.nodes[m.To].inputCh <- m
}
