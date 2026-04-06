package simulation

import (
	"fmt"
	pid "my-kad-dht/internal/id"
	"my-kad-dht/internal/network"
	config "my-kad-dht/internal/scenario"
	"path"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	ScenariosDir = "data/scenarios"
)

func Simulation(scenarioName string) {
	// logger
	log := zap.Must(zap.NewDevelopment())

	// scenario
	if !strings.HasSuffix(scenarioName, ".yaml") {
		scenarioName = fmt.Sprintf("%s.yaml", scenarioName)
	}
	scenario := config.MustLoad(path.Join(ScenariosDir, scenarioName))

	// init network: create bootstrap nodes
	net := network.New(scenario.Kademlia, scenario.BootstrapNodes)
	log.Info("create network",
		zap.Int("Boot nodes", len(scenario.BootstrapNodes)),
		zap.Int("Connections", len(scenario.Nodes[0].BootstrapVia)),
	)

	// start network (run bootstrap nodes)
	net.StartNetwork()
	log.Info("network started")

	// nodes join network
	nodes := net.CreateNNodes(scenario.Nodes, scenario.Kademlia)
	for i, joinInfo := range scenario.Nodes {
		net.Join(joinInfo)

		time.Sleep(10 * time.Millisecond)
		go func() {
			nodes[i].Run()
		}()
	}
	log.Info("nodes joined the network", zap.Int("count", len(scenario.Nodes)))

	// extra data structure for convenience
	nodeById := make(map[pid.PeerID]*network.Node)
	for _, node := range nodes {
		nodeById[node.ID()] = node
	}

	// starting work
	log.Info("doing workload... (publishing data)")

	//! now only publish data and then search it
	for _, action := range scenario.Workload {
		node := nodeById[pid.PeerID(action.Executor)]
		switch action.Type {
		case "store":
			node.Store(action.Key, action.Value)
		case "search":
			node.FindKey(action.Key)
		}
	}

	// save results
	net.DumpTopology()
	net.DumpMetrics()
}
