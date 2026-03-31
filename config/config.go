package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Network  Network  `yaml:"network"`
	Kademlia Kademlia `yaml:"kademlia"`
}

type Network struct {
	//? It can be polished to get into account all possible details but do we need it?
	Bootstrap Bootstrap `yaml:"bootstrap"`

	NodesCount struct {
		// nodes that act as usual "servers"
		common int
		// number of nodes which spread and retrieve (key, value) records across the network
		active int
	} `yaml:"nodes_count"`

	Workload struct {
		// "client" nodes that gonna publish and retrive records
		nodes int `yaml:"nodes"`
		// number of key-pairs to be published by "client" nodes
		records int `yaml:"records"`
	} `yaml:"workload"`
}

// Bootstrap 
type Bootstrap struct {
	NodesCount        int `yaml:"nodes_count"`
	Connections_count int `yaml:"connections_count"`
}

type Kademlia struct {
	K     int `yaml:"k"`     // bucket size
	Alpha int `yaml:"alpha"` // number of nodes to ask per hop
	Beta  int `yaml:"beta"`  // number of contacts to return after FIND_NODE, FIND_VALUE
}

func LoadConfig() *Config {
	path := flag.String("config-path", "", "")
	flag.Parse()

	if *path == "" {
		*path = os.Getenv("CONFIG_PATH")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(*path, &cfg); err != nil {
		log.Fatalf("failed to read config via path: %s", *path)
	}

	return &cfg
}
