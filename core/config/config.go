package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Seed uint64 `yaml:"seed"`

	Kademlia Kademlia   `yaml:"kademlia"`
	Network  P2pNetwork `yaml:"network"`
	Workload Workload   `yaml:"workload,omitempty"`
}

type Kademlia struct {
	BitSize int `yaml:"bit_size"`
	K       int `yaml:"k"`
	Alpha   int `yaml:"alpha"`
	Beta    int `yaml:"beta"`
}

type P2pNetwork struct {
	NodesCount int `yaml:"nodes_count"`

	JoinViaBootstrap bool `yaml:"join_via_bootstrap" env-default:"true"`
	Bootstrap_count  int  `yaml:"bootstrap_count"`
	Bootstrap_conns  int  `yaml:"bootstrap_conns"`
}

type Workload struct {
	Steps           int  `yaml:"steps"`
	Store           bool `yaml:"store"`
	LookupsPerStore int  `yaml:"lookups_per_store"`
}

const (
	configDir = "configs"
)

func MustLoad(configName string) *Config {
	if !strings.HasSuffix(configName, ".yaml") {
		configName = configName + ".yaml"
	}

	p := path.Join(configDir, configName)
	var c Config
	if err := cleanenv.ReadConfig(p, &c); err != nil {
		panic(fmt.Sprintf("failed to read config: %v", err))
	}

	return &c
}
