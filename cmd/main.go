package main

import (
	"fmt"
	pid "my-kad-dht/internal/id"
	"my-kad-dht/internal/network"
	"my-kad-dht/internal/storage"
)

func main() {
	test()
}

func test() {
	net := network.New()

	bootstrapNodesCount := 2
	bootstrapNodes := make([]*network.Node, bootstrapNodesCount)
	for i := range len(bootstrapNodes) {
		bootstrapNodes[i] = net.NewNode(
			pid.Generate(),
			storage.New(),
		)
	}
	net.AddBootstrapNodes(bootstrapNodes...)

	nodesCount := 10
	nodes := make([]*network.Node, nodesCount)
	for i := range len(nodes) {
		nodes[i] = net.NewNode(
			pid.Generate(),
			storage.New(),
		)
	}

	for _, node := range nodes {
		net.Join(node)
	}

	fmt.Println("It works")
}
