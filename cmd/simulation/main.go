package main

import (
	"my-kad-dht/internal/simulation"
	"os"
)

const (
	scenariosDir = "data/scenarios"
)

func main() {
	simulation.Simulation(os.Args[1])
}
