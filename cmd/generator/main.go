package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

const (
	scenariosDir = "data/scenarios"

	strLength = 10 // used for key, value for storing
)

type YAMLWriter struct {
	file *os.File
}

func NewYAMLWriter(outputPath string) *YAMLWriter {
	//! FILE NOT CLOSED
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	return &YAMLWriter{file: file}
}

func (w *YAMLWriter) WriteSection(name string, data any) {
	raw, _ := yaml.Marshal(data)
	fmt.Fprintf(w.file, "\n%s:\n", name)

	for _, line := range strings.Split(string(raw), "\n") {
		if line != "" {
			fmt.Fprintf(w.file, "  %s\n", line)
		}
	}
}

func (w *YAMLWriter) Write(text string) {
	_, err := w.file.Write([]byte(text))
	if err != nil {
		log.Fatalf("failed to write text \"%s\": %v", text, err)
	}
}

type node struct {
	ID           string   `yaml:"id"`
	Address      string   `yaml:"address"`
	BootstrapVia []string `yaml:"bootstrap_via,omitempty"`
	JoinOrder    int      `yaml:"join_order,omitempty"`
}

type action struct {
	Step       int    `yaml:"step"`
	ActionType string `yaml:"action"`
	Executor   string `yaml:"executor"`
	Key        string `yaml:"key"`
	Value      string `yaml:"value,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: generator <config.yaml>")
	}
	configPath := os.Args[1]

	cfg := LoadConfig(configPath)

	configName := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
	outPath := filepath.Join(scenariosDir, configName+".yaml")

	generator := NewGenerator(cfg.Seed)
	writer := NewYAMLWriter(outPath)
	log.Printf("generating scenario from %s -> %s", configPath, outPath)

	// write seed
	writer.Write(fmt.Sprintf("seed: %d", cfg.Seed))

	// kademlia params
	writer.WriteSection("Kademlia", cfg.Kademlia)

	// bootstrap nodes
	n := cfg.Network.Bootstrap.NodesCount
	bootstrapNodes := make([]node, n)
	for i := range n {
		bootstrapNodes[i] = node{
			ID:      generator.ID(),
			Address: generator.Addr(),
		}
	}
	writer.WriteSection("bootstrap_nodes", bootstrapNodes)

	// nodes
	n = cfg.Network.NodesCount
	nodes := make([]node, n)
	for i := range n {
		iNode := node{
			ID:           generator.ID(),
			Address:      generator.Addr(),
			JoinOrder:    i + 1,
			BootstrapVia: generator.RandomNodes(bootstrapNodes, cfg.Network.Bootstrap.Connections_count),
		}
		nodes[i] = iNode
	}
	writer.WriteSection("nodes", nodes)

	// workload: publish & search
	n = cfg.Workload.Publications
	actions := make([]action, n*2)
	for i := range n {
		publ := action{
			Step:       i + 1,
			ActionType: "store",
			Key:        generator.RandString(strLength),
			Value:      generator.RandString(strLength),
			Executor:   generator.RandomNode(nodes),
		}
		actions[i] = publ
	}

	for i := range n {
		search := action{
			Step:       i + 1 + n,
			ActionType: "search",
			Key:        actions[i].Key,
			Value:      actions[i].Value,
			Executor:   generator.RandomNode(nodes),
		}
		actions[i+n] = search
	}
	writer.WriteSection("workload", actions)
}

/* EXAMPLE INPUT

seed: 42

kademlia:
  bit_size: 160
  k:        10
  alpha:    3
  beta:     3

network:
  nodes_count: 1000
  bootstrap:
    nodes_count:       5
    connections_count: 2

workload:
  publications: 80

*/

type Config struct {
	Seed     uint64   `yaml:"seed" env-default:"42"`
	Network  Network  `yaml:"network"`
	Kademlia Kademlia `yaml:"kademlia"`
	Workload Workload `yaml:"workload"`
}

type Network struct {
	Bootstrap  Bootstrap `yaml:"bootstrap"`
	NodesCount int       `yaml:"nodes_count"` // total working nodes
}

type Bootstrap struct {
	NodesCount        int `yaml:"nodes_count"`
	Connections_count int `yaml:"connections_count"` // how many bootstrap nodes connect (out of NodesCount)
}

type Kademlia struct {
	BitSize int `yaml:"bit_size" env-default:"160"` // number of bits in ID
	K       int `yaml:"k"`                          // bucket size
	Alpha   int `yaml:"alpha"`                      // number of async requests to send in parallel during node lookup
	Beta    int `yaml:"beta"`                       // number of contacts to return in response for FindNode, FindValue
}

type Workload struct {
	Publications int `yaml:"publications"`
}

func validate(cfg *Config) {
	b := cfg.Network.Bootstrap
	b.Connections_count = min(b.Connections_count, b.NodesCount)
	cfg.Network.Bootstrap = b
}

func LoadConfig(path string) *Config {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatalf("failed to read config via path %s: %v", path, err.Error())
	}
	validate(&cfg)
	return &cfg
}

func generateID(rng *rand.Rand) string {
	buf := make([]byte, 20)
	for i := range buf {
		buf[i] = byte(rng.UintN(256))
	}
	return fmt.Sprintf("%x", buf)
}

func hashKey(key string) string {
	hash := sha1.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// Random generator

type Generator struct {
	seed  uint64
	rng   *rand.Rand
	faker *gofakeit.Faker
}

func NewGenerator(seed uint64) *Generator {
	rng := rand.New(rand.NewPCG(seed, 0))
	faker := gofakeit.NewFaker(
		gofakeit.NewFaker(rng, false),
		false,
	)

	return &Generator{
		seed:  seed,
		rng:   rng,
		faker: faker,
	}
}

func (g *Generator) ID() string {
	buf := make([]byte, 20)
	for i := range buf {
		buf[i] = byte(g.rng.UintN(256))
	}
	return fmt.Sprintf("%x", buf)
}

func (g *Generator) Addr() string {
	return fmt.Sprintf(
		"%s:%d",
		g.faker.IPv4Address(),
		g.rng.IntN(65535-1024)+1024,
	)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func (g *Generator) RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[g.rng.IntN(len(letterRunes))]
	}
	return string(b)
}

func (g *Generator) RandomNodes(slice []node, n int) []string {
	if n <= 0 {
		return []string{}
	}

	if len(slice) < n {
		return []string{}
	}

	shuffled := make([]node, len(slice))
	copy(shuffled, slice)

	g.rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	bootstrapNodes := shuffled[:n]
	bootstrapIds := make([]string, 0, n)
	for _, node := range bootstrapNodes {
		bootstrapIds = append(bootstrapIds, node.ID)
	}
	return bootstrapIds
}

func (g *Generator) RandomNode(slice []node) string {
	total := len(slice)
	return slice[g.rng.IntN(total)].ID
}
