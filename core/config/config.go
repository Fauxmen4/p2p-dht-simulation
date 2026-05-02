package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Seed     uint64     `yaml:"seed"`
	Kademlia Kademlia   `yaml:"kademlia"`
	Network  P2pNetwork `yaml:"network"`
	Latency  Latency    `yaml:"latency,omitempty"`
	Workload Workload   `yaml:"workload,omitempty"`
}

type Kademlia struct {
	BitSize int `yaml:"bit_size"`
	K       int `yaml:"k"`
	Alpha   int `yaml:"alpha"`
	Beta    int `yaml:"beta"`

	PeerDiversity bool `yaml:"peer_diversity"`
}

type P2pNetwork struct {
	NodesCount int `yaml:"nodes_count"`

	DropRate float64 `yaml:"drop_rate" env-default:"0"` // probability [0, 1) of dropping any single message

	// Bootstrap
	JoinViaBootstrap bool `yaml:"join_via_bootstrap" env-default:"true"`
	Bootstrap_count  int  `yaml:"bootstrap_count"`
	Bootstrap_conns  int  `yaml:"bootstrap_conns"`

	Latency Latency `yaml:"latency"`
}

type Latency struct {
	AreaSize       float64 `yaml:"area_size" env-default:"85.0"` // AreaSize of 2D square: [0, AreaSize] x [0. AreaSize] in ms.
	ServerFraction float64 `yaml:"server_fraction" env-default:"0.3"`
	ServerMean     float64 `yaml:"server_mean" env-default:"0.5"`
	ServerStd      float64 `yaml:"server_std" env-default:"0.3"`
	HomeMean       float64 `yaml:"home_mean" env-default:"6.0"`
	HomeStd        float64 `yaml:"home_std" env-default:"3.5"`
	MinHeight      float64 `yaml:"min_height" env-default:"0.1"`
}

type Workload struct {
	Churn Churn `yaml:"churn,omitempty"`

	Steps           int  `yaml:"steps"`
	Store           bool `yaml:"store"`
	LookupsPerStore int  `yaml:"lookups_per_store"`
}

type Churn struct {
	Phase  string  `yaml:"phase"`  // before_lookup | before_search
	Lambda float64 `yaml:"lambda"` // how many nodes leave/join during step
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
