package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Network  Network  `yaml:"network"`
	Kademlia Kademlia `yaml:"kademlia"`
}

type Network struct {
	Bootstrap  Bootstrap `yaml:"bootstrap"`
	NodesCount int       `yaml:"nodes_count"` // total working nodes
}

type Bootstrap struct {
	NodesCount        int `yaml:"nodes_count"`
	Connections_count int `yaml:"connections_count" env-default:"1"` // how many bootstrap nodes connect (out of NodesCount)
}

type Kademlia struct {
	BitSize int `yaml:"bit_size" env-default:"160"` // number of bits in ID
	K       int `yaml:"k"`                          // bucket size
	Alpha   int `yaml:"alpha"`                      // number of async requests to send in parallel during node lookup
	Beta    int `yaml:"beta"`                       // number of contacts to return in response for FindNode, FindValue
}

func validate(cfg *Config) {
	b := cfg.Network.Bootstrap
	b.Connections_count = min(b.Connections_count, b.NodesCount)
	cfg.Network.Bootstrap = b
}

func LoadConfig(path string) *Config {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatalf("failed to read config via path: %s", path)
	}

	validate(&cfg)

	return &cfg
}
