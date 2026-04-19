package main

import (
	"my-kad-dht/core/simulation"
	"os"
)

func main() {
	configName := os.Args[1]
	simulation.ConfigBased(configName)
}
