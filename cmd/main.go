// TODO: move this code to simulation
// ? Currently I am working on static scenario
package main

import (
	"fmt"
	"my-kad-dht/config"
	"my-kad-dht/internal/network"
)

func main() {
	// read config
	cfg := config.LoadConfig("config/example.yaml")

	// init network
	net := network.New(*cfg)

	fmt.Println(net)
	_ = net
}
