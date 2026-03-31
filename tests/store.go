package tests

import (
	"my-kad-dht/config"
	pid "my-kad-dht/internal/id"
	"my-kad-dht/internal/network"
	"my-kad-dht/internal/storage"
	"time"
)

func Test_AddRecords() {
	config := config.Config{
		Network: config.Network{},
		Kademlia: config.Kademlia{
			K:          20,
			Alpha:      3,
		},
	}

	net := network.New(config)

	bootstrapNodesCount := 5
	nodesCount := 20
	clientNodesCount := 3
	recordRounds := 5

	bootstrapNodes := make([]*network.Node, bootstrapNodesCount)
	for i := range len(bootstrapNodes) {
		bootstrapNodes[i] = net.NewNode(
			pid.Generate(),
			storage.New(),
		)
	}
	net.AddBootstrapNodes(bootstrapNodes...)

	net.StartNetwork()

	nodes := make([]*network.Node, nodesCount)
	for i := range len(nodes) {
		nodes[i] = net.NewNode(
			pid.Generate(),
			storage.New(),
		)
	}

	for _, node := range nodes {
		// fmt.Printf("Node %s is trying to join the network\n", node.ID())
		net.Join(node)

		// fmt.Printf("Node %s successfully joined the network\n", node.ID())
		// node.RoutingTable.Print()

		time.Sleep(1 * time.Millisecond)
		go func() {
			node.Run()
		}()
	}

	// TESTING
	clientNodes := make([]*network.Node, clientNodesCount)
	for i := range len(clientNodes) {
		clientNodes[i] = net.NewNode(
			pid.Generate(),
			storage.New(),
		)
	}

	for _, node := range clientNodes {
		net.Join(node)

		for range recordRounds {
			node.StoreRandStr()
		}

		go func() {
			node.Run()
		}()
	}

	time.Sleep(5 * time.Second)

	for _, node := range nodes {
		node.KVStorage.Print()
	}
}
