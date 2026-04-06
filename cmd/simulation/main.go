package main

import (
	"my-kad-dht/core/simulation"
	"os"
)

const (
	scenariosDir = "data/scenarios"
)

func main() {
	simulation.Simulation(os.Args[1])
}
