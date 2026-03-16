package network

import (
	"encoding/json"
	"fmt"
)

// DumpTopology creates .json with network description in order to visualize it with vis.js.
// TODO: optimize it as hard as i can
func (n *Network) DumpTopology() {
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

	dump, err := json.Marshal(data)
	fmt.Println(err)
	fmt.Println(string(dump))
}
