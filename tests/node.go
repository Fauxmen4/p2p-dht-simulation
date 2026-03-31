package tests

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
	rt "my-kad-dht/internal/table"
)

func Test_NodeCreation() {
	nodeID := pid.Generate()

	table := rt.NewRoutingTable(20, 160, nodeID)

	table.Print()

	testNodeIds := make([]pid.PeerID, 0, 10)
	for range 20 {
		testNodeIds = append(testNodeIds, pid.Generate())
	}

	for _, id := range testNodeIds {
		table.Add(id, addr.GenerateAddr())
	}

	table.Print()
}
