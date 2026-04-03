package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Network  Network  `yaml:"network"`
	Kademlia Kademlia `yaml:"kademlia"`
	Workload Workload `yaml:"workload"`
}

type Network struct {
	Bootstrap  Bootstrap `yaml:"bootstrap"`
	NodesCount int       `yaml:"nodes_count"` // total working nodes
	DropRate   float64   `yaml:"drop_rate" env-default:"0.0"`   // probability [0.0, 1.0] of dropping a message
	LatencyMs  int       `yaml:"latency_ms"`  // delivery delay in milliseconds
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

type Workload struct {
	Publications int `yaml:"publications"`
	KeySize      int `yaml:"key_size" env-default:"10"`
	ValueSize    int `yaml:"value_size" env-default:"10"`
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
