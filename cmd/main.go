package main

import (
	"fmt"
	"math/rand/v2"
	"my-kad-dht/internal/config"
	"my-kad-dht/internal/network"
	"my-kad-dht/internal/utils"
	"time"

	"go.uber.org/zap"
)

func main() {
	// logger
	log := zap.Must(zap.NewDevelopment())

	// read config
	cfg := config.LoadConfig("configs/static_success.yaml")

	// init network
	net := network.New(*cfg)
	log.Info("create network",
		zap.Int("Boot nodes", cfg.Network.Bootstrap.NodesCount),
		zap.Int("Connections", cfg.Network.Bootstrap.Connections_count),
	)

	// start network (enable bootstrap)
	net.StartNetwork()
	log.Info("network started")

	// nodes join network
	nodes := net.CreateNNodes(cfg.Network.NodesCount)
	for _, node := range nodes {
		net.Join(node)

		time.Sleep(1 * time.Millisecond)
		go func() {
			node.Run()
		}()
	}
	log.Info("nodes joined the network",
		zap.Int("count", cfg.Network.NodesCount),
	)

	// starting work
	log.Info("doing workload... (publishing data)")

	// publish data
	count := cfg.Workload.Publications
	data := make(map[string]string, count)
	for range count {
		key := utils.RandString(cfg.Workload.KeySize)
		value := utils.RandString(cfg.Workload.ValueSize)
		data[key] = value
	}

	for key, value := range data {
		index := rand.IntN(cfg.Network.NodesCount)
		nodes[index].Store(key, value)

		time.Sleep(5 * time.Millisecond)
	}
	log.Info("data successfully published")

	// search data
	log.Info("searching data...")
	for key, _ := range data {
		index := rand.IntN(cfg.Network.NodesCount)
		_, _ = nodes[index].FindKey(key)

		time.Sleep(5 * time.Millisecond)
	}
	log.Info("finished searching")

	// print out results
	for _, node := range nodes {
		info := node.Metrics.SearchHistory()
		if len(info) != 0 {
			fmt.Println(info)
		}
	}

	// save results
	net.DumpTopology()
	net.DumpMetrics()
}
