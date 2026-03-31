package tests

import (
	"fmt"
	"math/rand"
	"my-kad-dht/config"
	"my-kad-dht/internal/network"
	"my-kad-dht/internal/utils"
	"time"
)

func Test_RandomWorkload() {
	config := config.Config{
		Network: config.Network{
			Bootstrap: config.Bootstrap{
				Connections_count: 3,
			},
		},
		Kademlia: config.Kademlia{
			BucketSize: 10,
			K:          10,
			Alpha:      2,
		},
	}

	net := network.New(config)

	bootstrapNodesCount := 4
	nodesCount := 200

	opsCount := 20 // workload size

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

	// Generate random workload
	ops := make([]Operation, 0, opsCount)
	keys := make([]string, 0)

	for range opsCount {
		if rand.Float64() < 0.5 || len(keys) == 0 {
			key, value := utils.RandString(8), utils.RandString(8)
			ops = append(ops, Operation{
				Type:  "PUT",
				Key:   key,
				Value: value,
			})
			keys = append(keys, key)
		} else {
			ops = append(ops, Operation{
				Type: "GET",
				Key:  keys[rand.Intn(len(keys))],
			})
		}
	}

	activeNodes := net.CreateNNodes(opsCount)
	for i, node := range activeNodes {
		net.Join(node)

		op := ops[i]
		switch op.Type {
		case "PUT":
			node.Store(op.Key, op.Value)
		case "GET":
			node.FindKey(op.Key)
		}

		fmt.Println("Statistics")
		fmt.Println(node.Metrics.SearchInfo())

		time.Sleep(1 * time.Millisecond)
		go func() {
			node.Run()
		}()
	}
}

type Operation struct {
	Type  string
	Key   string
	Value string
}
