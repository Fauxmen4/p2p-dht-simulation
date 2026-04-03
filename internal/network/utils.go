package network

import (
	"encoding/json"
	"fmt"
	"log"
	pid "my-kad-dht/internal/id"
	strg "my-kad-dht/internal/storage"
	"os"
	"strings"
	"time"
)

// DumpTopology creates .json with network description in order to visualize it with vis.js.
// TODO: optimize it as hard as i can
func (n *Network) DumpTopology(outputStream string) {
	data := make(map[string]any, 0)

	set := make(map[string]struct{})
	bootstrapIds := make([]string, 0)
	for _, node := range n.bootstrapNodes {
		bootstrapIds = append(bootstrapIds, string(node.id))
		set[string(node.id)] = struct{}{}
	}
	data["bootstrap_nodes"] = bootstrapIds // for highlighting on graph

	nodeIds := make([]string, 0)
	for _, node := range n.nodes {
		if _, ok := set[string(node.id)]; !ok {
			nodeIds = append(nodeIds, string(node.id))
		}
	}
	data["nodes"] = nodeIds

	added := make(map[[2]string]struct{})
	edges := make([][2]string, 0)
	for _, node := range n.nodes {
		connectedIds := node.RoutingTable.ReturnAllIds()
		for _, id := range connectedIds {
			edge := [2]string{string(node.id), string(id)}
			if _, ok := added[edge]; !ok {
				edges = append(edges, edge)
				added[edge] = struct{}{}
				added[[2]string{edge[1], edge[0]}] = struct{}{}
			}
		}
	}
	data["edges"] = edges

	out := os.Stdout
	if strings.ToLower(outputStream) != "stdout" {
		dumpName := fmt.Sprintf("data/topology/%v.json", time.Now())
		file, err := os.OpenFile(dumpName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("failed to create dump file: %v", err.Error())
		}
		out = file
	}

	dump, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("failed to write network dump: %v", err.Error())
	}
	out.Write(dump)

	// fmt.Println(err)
	// fmt.Printf(string(dump))
}

func (n *Node) DumpStorage() {
	fmt.Printf("Storage of node: %s", n.ID())
	n.KVStorage.Print()
}

func (n *Network) CreateNNodes(count int) []*Node {
	nodes := make([]*Node, count)
	for i := range len(nodes) {
		nodes[i] = n.NewNode(
			pid.Generate(),
			strg.New(),
		)
	}
	return nodes
}
