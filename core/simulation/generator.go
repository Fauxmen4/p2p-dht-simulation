package simulation

import (
	"fmt"
	"math/rand/v2"
	"my-kad-dht/core/addr"
	"my-kad-dht/core/config"
	cfg "my-kad-dht/core/config"
	pid "my-kad-dht/core/id"

	"github.com/brianvoe/gofakeit/v7"
	"gonum.org/v1/gonum/stat/distuv"
)

// Generator is a deterministic source of randomness
type generator struct {
	seed    uint64
	rng     *rand.Rand
	faker   *gofakeit.Faker
	poisson distuv.Poisson

	bootstrapIDs []pid.PeerID
}

// Gnerator constructor
func NewGenerator(config *cfg.Config) *generator {
	rng := rand.New(rand.NewPCG(config.Seed, 0))
	faker := gofakeit.NewFaker(
		gofakeit.NewFaker(rng, false),
		false,
	)

	poisson := distuv.Poisson{}
	if lambda := config.Workload.Churn.Lambda; lambda != 0 {
		poisson = distuv.Poisson{Lambda: lambda, Src: rng}
	}

	return &generator{
		seed:         config.Seed,
		rng:          rng,
		faker:        faker,
		bootstrapIDs: []pid.PeerID{},
		poisson:      poisson,
	}
}

// Generate random ID
func (g *generator) id() pid.PeerID {
	buf := make([]byte, 20)
	for i := range buf {
		buf[i] = byte(g.rng.UintN(256))
	}
	return pid.PeerID(fmt.Sprintf("%x", buf))
}

// Generate random peer address
func (g *generator) addr() addr.Addr {
	return addr.Addr(fmt.Sprintf(
		"%s:%d",
		g.faker.IPv4Address(),
		g.rng.IntN(65535-1024)+1024,
	))
}

// Return n random elements from the given slice
func randomN[T any](rng *rand.Rand, slice []T, n int) []T {
	if n <= 0 || len(slice) == 0 {
		return []T{}
	}
	if n >= len(slice) {
		return slice
	}
	shuffled := make([]T, len(slice))
	copy(shuffled, slice)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:n]
}

// Return n random IDs of bootstrap nodes
func (g *generator) randomBootstrapIDs(n int) []pid.PeerID {
	return randomN(g.rng, g.bootstrapIDs, n)
}

// Add bootstrap node ID to internal ID storage.
// It will be used in randomBootstrapIDs
func (g *generator) addBootstrapID(id pid.PeerID) {
	g.bootstrapIDs = append(g.bootstrapIDs, id)
}

// Generate random data for n bootstrap nodes.
func (g *generator) nBootstrapNodes(n int) []cfg.NodeSpec {
	nodes := make([]cfg.NodeSpec, 0, n)
	for range n {
		ID := g.id()
		nodes = append(nodes, cfg.NodeSpec{
			ID:   ID,
			Addr: g.addr(),
		})
		g.bootstrapIDs = append(g.bootstrapIDs, ID)
	}

	return nodes
}

// Generate random data for number of nodes from config.
// Differs from nBootstrapNodes by bootstrap nodes choice.
func (g *generator) nNewNodes(cfg config.P2pNetwork) []cfg.NodeSpec {
	nodes := make([]config.NodeSpec, cfg.NodesCount)
	for i := range cfg.NodesCount {
		ID := g.id()
		nodes[i] = config.NodeSpec{
			ID:           ID,
			Addr:         g.addr(),
			BootstrapVia: g.randomBootstrapIDs(cfg.Bootstrap_conns),
		}
		// if !cfg.JoinViaBootstrap {
		// 	g.addBootstrapID(ID)
		// }
	}
	return nodes
}

func (g *generator) newNode(cfg config.P2pNetwork) config.NodeSpec {
	return config.NodeSpec{
		ID:           g.id(),
		Addr:         g.addr(),
		BootstrapVia: g.randomBootstrapIDs(cfg.Bootstrap_conns),
	}
}

// Publish data generation

const (
	keySize   = 8
	valueSize = 8
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func (g *generator) randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[g.rng.IntN(len(letterRunes))]
	}
	return string(b)
}

func (g *generator) randKV() (string, string) {
	return g.randString(keySize), g.randString(valueSize)
}

// Poisson distribution sampling

func (g *generator) isChurn() bool {
	return g.poisson != distuv.Poisson{}
}

func (g *generator) poissonSample() int {
	return int(g.poisson.Rand())
}
