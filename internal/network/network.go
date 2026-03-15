package network

import (
	"math/rand/v2"
	"my-kad-dht/config"
	"my-kad-dht/internal/addr"
	msg "my-kad-dht/internal/message"
)

type Network struct {
	config config.Config

	nodes map[addr.Addr]*Node

	bootstrapNodes []*Node
}

func New(cfg config.Config) *Network {
	net := &Network{
		config: cfg,

		nodes:          make(map[addr.Addr]*Node),
		bootstrapNodes: make([]*Node, 0),
	}

	return net
}

func (n *Network) AddBootstrapNodes(nodes ...*Node) {
	for _, node := range nodes {
		n.nodes[node.addr] = node
	}
	n.bootstrapNodes = append(n.bootstrapNodes, nodes...)
}

// StartNetwork runs all bootstrap nodes in separate goroutines
// ! CONCURRENT
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

// SendBlocking sends message and blocks until reader appears
func (n *Network) SendBlocking(msg msg.Message) {
	n.nodes[msg.Receiver()].inputCh <- msg
}

func (n *Network) Join(node *Node) {
	// register node in network
	n.nodes[node.addr] = node

	// choose one bootstrap node randomly
	index := rand.IntN(len(n.bootstrapNodes))
	bootstrapNode := n.bootstrapNodes[index]

	// join the network through node lookup
	node.Join(bootstrapNode.id, bootstrapNode.addr)
}
