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
	Boostrap struct {
		nodesCount        int `yaml:"nodes_count"`
		connections_count int `yaml:"connections_count"`
	} `yaml:"bootstrap"`

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

type Kademlia struct {
	BucketSize int `yaml:"bucket_size"` //? I can't understand is it different from K?
	K          int `yaml:"k"`
	Alpha      int `yaml:"alpha"`
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
