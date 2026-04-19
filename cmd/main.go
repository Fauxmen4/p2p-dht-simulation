package main

import (
	"my-kad-dht/core/simulation"
	"os"
)

const (
	scenariosDir = "data/scenarios"
)

func main() {
	simulation.ConfigBased(os.Args[1])
}
