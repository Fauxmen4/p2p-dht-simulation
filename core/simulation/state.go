package simulation

import (
	"context"
	"fmt"
	"my-kad-dht/core/addr"
	"my-kad-dht/core/config"
	"my-kad-dht/core/network"
	"my-kad-dht/core/node"
	"time"
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

	// publish
	key, value := s.gen.randKV()
	s.kvData = append(s.kvData, [2]string{key, value})
	publisher := randomN(s.gen.rng, s.nodes, 1)[0]
	publisher.Store(context.Background(), key, value)

	// b). churn
	if s.gen.isChurn() && s.cfg.Workload.Churn.Phase == "before_search" {
		s.applyChurn()
	}

	// fmt.Println("seacrhing", key) //! LOGGING

	// choose nodes
	searchers := randomN(s.gen.rng, s.nodes, s.cfg.Workload.LookupsPerStore)
	// search
	for i := range searchers {
		searchers[i].ValueLookup(context.Background(), key)
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

	// fmt.Println(k, "nodes left") //! LOGGING

	start := time.Now()

	// joining
	for range k {
		spec := s.gen.newNode()
		newNode := s.net.AddAndJoin(spec, s.cfg.Kademlia)
		s.nodes = append(s.nodes, newNode)
	}

	fmt.Println(k, "nodes joined", "after", time.Since(start).Seconds()) //! LOGGING

}
