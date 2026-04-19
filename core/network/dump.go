package network

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"sort"
	"strings"
	"time"
)

// DumpTopology creates .json with network description in order to visualize it with vis.js.
// TODO: optimize it
func (n *Network) DumpTopology() {
	data := make(map[string]any, 0)

	set := make(map[string]struct{})
	bootstrapIds := make([]string, 0)
	for _, node := range n.bootstrapNodes {
		bootstrapIds = append(bootstrapIds, string(node.ID()))
		set[string(node.ID())] = struct{}{}
	}
	data["bootstrap_nodes"] = bootstrapIds // for highlighting on graph

	nodeIds := make([]string, 0)
	for _, node := range n.nodes {
		if _, ok := set[string(node.ID())]; !ok {
			nodeIds = append(nodeIds, string(node.ID()))
		}
	}
	data["nodes"] = nodeIds

	added := make(map[[2]string]struct{})
	edges := make([][2]string, 0)
	for _, node := range n.nodes {
		connectedIds := node.RoutingTable.ReturnAllIds()
		for _, id := range connectedIds {
			edge := [2]string{string(node.ID()), string(id)}
			if _, ok := added[edge]; !ok {
				edges = append(edges, edge)
				added[edge] = struct{}{}
				added[[2]string{edge[1], edge[0]}] = struct{}{}
			}
		}
	}
	data["edges"] = edges

	if err := dumpToJSON("topology", data); err != nil {
		log.Printf("faied to dump network topology: %v", err.Error())
	}
}

func (n *Network) DumpMetrics() {
	data := make(map[string]any, 0)

	handledRPCs := []int{}
	sentRPCs := []int{}

	hopsCount := []int{}
	success := 0
	total := 0

	for _, node := range n.bootstrapNodes {
		handledRPCs = append(handledRPCs, node.Metrics.HandledRPCs())
		sentRPCs = append(sentRPCs, node.Metrics.SentRPCs())

		successHopCount := node.Metrics.SuccessHopCount()
		success += len(successHopCount)
		total += node.Metrics.CountKeyLookups()
		hopsCount = append(hopsCount, successHopCount...)
	}
	for _, node := range n.nodes {
		handledRPCs = append(handledRPCs, node.Metrics.HandledRPCs())
		sentRPCs = append(sentRPCs, node.Metrics.SentRPCs())

		successHopCount := node.Metrics.SuccessHopCount()
		success += len(successHopCount)
		total += node.Metrics.CountKeyLookups()
		hopsCount = append(hopsCount, successHopCount...)
	}

	var sum float64 = 0
	for _, hops := range hopsCount {
		sum += float64(hops)
	}
	var avg float64 = sum / float64(len(hopsCount))
	fmt.Println(avg, len(hopsCount))
	fmt.Println("Median:", median(hopsCount))
	fmt.Println("Percentile 0.95:", percentile(hopsCount, 0.95))
	fmt.Println("Percentile 0.99:", percentile(hopsCount, 0.99))

	sum = 0
	for _, handled := range handledRPCs {
		sum += float64(handled)
	}
	fmt.Println("handled RPCs:",
		slices.Max(handledRPCs),
		float64(sum)/float64(len(handledRPCs)),
	)

	fmt.Println("Success rate:", fmt.Sprintf("%d/%d", success, total))

	data["handled_rpcs"] = handledRPCs
	data["sent_rpcs"] = sentRPCs
	data["key_lookups"] = map[string]any{
		"success_hops_count": hopsCount,
		"total":              total,
		"success":            success,
		"success_rate":       success / total,
	}

	if err := dumpToJSON("metrics", data); err != nil {
		log.Printf("failed to dump metrics: %v", err.Error())
	}
}

func dumpToJSON(dir string, data map[string]any) error {
	dumpName := strings.ReplaceAll(
		fmt.Sprintf("data/%s/%v.json", dir, time.Now()),
		" ",
		"_",
	)
	file, err := os.OpenFile(dumpName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create dump file: %v", err.Error())
	}

	dump, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize dump into .json format: %v", err.Error())
	}

	_, err = file.Write(dump)
	if err != nil {
		return fmt.Errorf("failed to write dump: %v", err.Error())
	}

	return nil
}

func median(data []int) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}

	sorted := make([]int, n)
	copy(sorted, data)
	sort.Ints(sorted)

	if n%2 == 0 {
		return float64(sorted[n/2-1]+sorted[n/2]) / 2
	}
	return float64(sorted[n/2])
}

func percentile(data []int, p float64) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}

	sorted := make([]int, n)
	copy(sorted, data)
	sort.Ints(sorted)

	idx := int(float64(n-1) * p)
	return float64(sorted[idx])
}
