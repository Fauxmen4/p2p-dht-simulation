package simulation

import (
	"context"
	"fmt"
	"my-kad-dht/core/config"
	"my-kad-dht/core/network"
	"time"

	"go.uber.org/zap"
)

func ConfigBased(configName string) {
	// init logger
	log := zap.Must(zap.NewDevelopment())

	// config
	cfg := config.MustLoad(configName)
	log.Info("loaded config", zap.String("config_name", configName))

	// deterministic random generator
	gen := NewGenerator(cfg)

	// determine bootstrap nodes
	bootstrapNodesSpec := gen.nBootstrapNodes(cfg)

	// create network
	net := network.New(*cfg, bootstrapNodesSpec)
	log.Info("create network",
		zap.Int("Boot nodes", cfg.Network.Bootstrap_count),
		zap.Int("Connections", cfg.Network.Bootstrap_conns),
		zap.Bool("Only via bootstrap", cfg.Network.JoinViaBootstrap),
	)

	// start network (run bootstrap nodes)
	net.StartNetwork()
	log.Info("network started")

	// determine nodes
	nodesSpec := gen.nNewNodes(cfg)

	// nodes join network
	nodes := net.CreateNNodes(nodesSpec, cfg.Kademlia)
	for i, joinInfo := range nodesSpec {
		// run node
		go func(idx int) {
			nodes[idx].Run(context.Background())
		}(i)
		// add routing table with init node data
		net.Join(joinInfo)
		time.Sleep(10 * time.Millisecond) //! give Run() time to start before Join sends RPCs

		// fmt.Println("joined node:", i)
	}
	log.Info("nodes joined the network", zap.Int("count", len(nodes)))

	// starting work
	log.Info("doing workload... (publishing data)")

	state := simState{
		cfg:    cfg,
		net:    net,
		nodes:  nodes,
		gen:    gen,
		kvData: [][2]string{},
	}

	for range cfg.Workload.Steps {
		state.step()
	}

	// save results
	net.DumpTopology()
	net.DumpMetrics()
	log.Info("topology & metrics were dumped")
}
