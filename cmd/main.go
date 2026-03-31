package main

import (
	"fmt"
	"my-kad-dht/config"
	"my-kad-dht/internal/network"
	"time"
)

func main() {
	config := config.Config{
		Network: config.Network{
			Bootstrap: config.Bootstrap{
				Connections_count: 2,
			},
		},
		Kademlia: config.Kademlia{
			BucketSize: 10,
			K:          10,
			Alpha:      3,
		},
	}

	net := network.New(config)

	bootstrapNodesCount := 5
	nodesCount := 1000

	bootstrapNodes := net.CreateNNodes(bootstrapNodesCount)
	net.AddBootstrapNodes(bootstrapNodes...)

	net.StartNetwork()

	nodes := net.CreateNNodes(nodesCount)

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

	// publish
	publishCount := 20
	rounds := 4
	keys := make([]string, 0, publishCount*rounds)

	publishNodes := net.CreateNNodes(publishCount)
	for _, node := range publishNodes {
		net.Join(node)

		for range rounds {
			key, value := node.StoreRandStr()
			_ = value
			keys = append(keys, key)
		}
		time.Sleep(1 * time.Millisecond)
		go func() {
			node.Run()
		}()
	}

	// search
	hops_count := make([]int, 0)
	searchNodes := net.CreateNNodes(publishCount * rounds)
	for i, node := range searchNodes {
		net.Join(node)

		key := keys[i]
		value, bool := node.FindKey(key)
		fmt.Println(value, bool, node.Metrics.SearchInfo())
		hops_count = append(hops_count, node.Metrics.ReturnHops()...)
		time.Sleep(1 * time.Millisecond)
		go func() {
			node.Run()
		}()
	}

	rpcCount := make([]int, 0, 1000)
	for _, node := range nodes {
		rpcCount = append(rpcCount, node.Metrics.CountRPCs())
	}

	fmt.Println(rpcCount)
	fmt.Println(hops_count)
}
