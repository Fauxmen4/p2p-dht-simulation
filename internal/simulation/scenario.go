package simulation

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Seed           uint64     `yaml:"seed"`
	Kademlia       Kademlia   `yaml:"Kademlia"`
	BootstrapNodes []NodeSpec `yaml:"bootstrap_nodes"`
	Nodes          []NodeSpec `yaml:"nodes"`
	Workload       []Action   `yaml:"workload"`
}

type Kademlia struct {
	BitSize int `yaml:"bit_size" env-default:"160"` // number of bits in ID
	K       int `yaml:"k"`                          // bucket size
	Alpha   int `yaml:"alpha"`                      // number of async requests to send in parallel during node lookup
	Beta    int `yaml:"beta"`                       // number of contacts to return in response for FindNode, FindValue
}

type Action struct {
	Step     int    `yaml:"step"`
	Type     string `yaml:"action"`
	Executor string `yaml:"executor"`
	Key      string `yaml:"key"`
	Value    string `yaml:"value,omitempty"`
}

type NodeSpec struct {
	ID           string   `yaml:"id"`
	Address      string   `yaml:"address"`
	BootstrapVia []string `yaml:"bootstrap_via,omitempty"`
	JoinOrder    int      `yaml:"join_order,omitempty"`
}

func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file %s: %w", path, err)
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal .yaml scenario %s: %w", path, err)
	}

	return &s, nil
}
