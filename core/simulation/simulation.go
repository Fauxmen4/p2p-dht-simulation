package simulation

import (
	"context"
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
	gen := NewGenerator(cfg.Seed)

	// determine bootstrap nodes
	bootstrapNodesSpec := gen.nBootstrapNodes(cfg.Network.Bootstrap_count)

	// create network
	net := network.New(cfg.Kademlia, bootstrapNodesSpec)
	log.Info("create network",
		zap.Int("Boot nodes", cfg.Network.Bootstrap_count),
		zap.Int("Connections", cfg.Network.Bootstrap_conns),
		zap.Bool("Only via bootstrap", cfg.Network.JoinViaBootstrap),
	)

	// start network (run bootstrap nodes)
	net.StartNetwork()
	log.Info("network started")

	// determine nodes
	nodesSpec := gen.nNodes(cfg.Network)

	// nodes join network
	nodes := net.CreateNNodes(nodesSpec, cfg.Kademlia)
	for i, joinInfo := range nodesSpec {
		go func(idx int) {
			nodes[i].Run(context.Background())
		}(i)
		time.Sleep(10 * time.Millisecond) //! give Run() time to start before Join sends RPCs
		net.Join(joinInfo)
	}
	log.Info("nodes joined the network", zap.Int("count", len(nodes)))

	// starting work
	log.Info("doing workload... (publishing data)")

	kvData := [][2]string{}
	for range cfg.Workload.Steps {
		// choose nodes for workload
		nActiveNodes := cfg.Workload.LookupsPerStore
		if cfg.Workload.Store {
			nActiveNodes += 1
		}
		activeNodes := randomN(gen.rng, nodes, nActiveNodes)

		// generate data
		key, value := gen.randKV()
		kvData = append(kvData, [2]string{key, value})

		// publish/search itself
		activeNodes[0].Store(context.Background(), key, value)
		for i := 1; i < len(activeNodes); i++ {
			activeNodes[i].ValueLookup(context.Background(), key)
		}
	}

	// save results
	net.DumpTopology()
	net.DumpMetrics()
	log.Info("topology & metrics were dumped")
}
