package simulation

import (
	"context"
	"fmt"
	"my-kad-dht/core/addr"
	"my-kad-dht/core/config"
	"my-kad-dht/core/network"
	"my-kad-dht/core/node"
	"time"

	"go.uber.org/zap"
)

type simState struct {
	cfg    *config.Config
	net    *network.Network
	nodes  []*node.Node
	gen    *generator
	kvData [][2]string
}

func (s *simState) step() {
	// a). churn
	if s.gen.isChurn() && s.cfg.Workload.Churn.Phase == "before_lookup" {
		s.applyChurn()
	}

	// workload
	// choose nodes for workload
	nWorkingNodes := s.cfg.Workload.LookupsPerStore
	if s.cfg.Workload.Store {
		nWorkingNodes += 1
	}
	workingNodes := randomN(s.gen.rng, s.nodes, nWorkingNodes)

	// publish
	key, value := s.gen.randKV()
	s.kvData = append(s.kvData, [2]string{key, value})
	workingNodes[0].Store(context.Background(), key, value)

	// b). churn
	if s.gen.isChurn() && s.cfg.Workload.Churn.Phase == "before_search" {
		s.applyChurn()
	}

	fmt.Println("seacrhing")

	// search
	for i := 1; i < len(workingNodes); i++ {
		workingNodes[i].ValueLookup(context.Background(), key)
	}
}

func (s *simState) applyChurn() {
	k := min(len(s.nodes), s.gen.poissonSample()) // number of nodes to leave/join

	// leaving
	leaving := randomN(s.gen.rng, s.nodes, k)
	leavingSet := make(map[addr.Addr]struct{}, k)
	for _, node := range leaving {
		s.net.Remove(node)
		leavingSet[node.Addr()] = struct{}{}
	}

	survived := s.nodes[:0]
	for _, node := range s.nodes {
		if _, gone := leavingSet[node.Addr()]; !gone {
			survived = append(survived, node)
		}
	}
	s.nodes = survived

	fmt.Println(k, "nodes left")

	start := time.Now()

	// joining
	for range k {
		spec := s.gen.newNode(s.cfg.Network)
		newNode := s.net.AddAndJoin(spec, s.cfg.Kademlia)
		s.nodes = append(s.nodes, newNode)
	}

	fmt.Println(k, "nodes joined", "after", time.Since(start).Seconds())

}

func ConfigBased(configName string) {
	// init logger
	log := zap.Must(zap.NewDevelopment())

	// config
	cfg := config.MustLoad(configName)
	log.Info("loaded config", zap.String("config_name", configName))

	// deterministic random generator
	gen := NewGenerator(cfg)

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
	nodesSpec := gen.nNewNodes(cfg.Network)

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
