package node

import "fmt"

func (n *Node) DumpStorage() {
	fmt.Printf("Storage of node: %s", n.ID())
	n.KVStorage.Print()
}
