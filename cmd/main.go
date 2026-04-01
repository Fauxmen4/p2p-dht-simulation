// TODO: move this code to simulation
// ? Currently I am working on static scenario
package main

import (
	"my-kad-dht/config"
	"my-kad-dht/internal/network"
	"time"

	"go.uber.org/zap"
)

func main() {
	// logger
	log := zap.Must(zap.NewDevelopment())

	// read config
	cfg := config.LoadConfig("config/example.yaml")

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

		time.Sleep(1 * time.Millisecond) // TODO: WTF?
		go func() {
			node.Run()
		}()
	}
	log.Info("nodes joined the network",
		zap.Int("count", cfg.Network.Bootstrap.NodesCount),
	)

	for _, node := range nodes {
		node.RoutingTable.Print()
	}
}
