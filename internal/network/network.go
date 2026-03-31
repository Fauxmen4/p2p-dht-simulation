package network

import (
	"math/rand/v2"
	"my-kad-dht/config"
	"my-kad-dht/internal/addr"
	msg "my-kad-dht/internal/message"
)

type Network struct {
	config         config.Config       // configuration of everything
	nodes          map[addr.Addr]*Node // map with all nodes, used to address messages from one node to another
	bootstrapNodes []*Node             // nodes for joining the network
}

// New is network constructor.
// Returns empty network (no nodes), only config is added.
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
	bootstrapNodes := n.SelectBootstrap(n.config.Network.Bootstrap.Connections_count)
	for _, bNode := range bootstrapNodes {
		// join the network through node lookup
		node.Join(bNode.id, bNode.addr)
	}
}

// SelectBootstrap returns n random bootsrap nodes
func (n *Network) SelectBootstrap(count int) []*Node {
	shuffled := make([]*Node, len(n.bootstrapNodes))
	copy(shuffled, n.bootstrapNodes)

	rand.Shuffle(count, func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}
