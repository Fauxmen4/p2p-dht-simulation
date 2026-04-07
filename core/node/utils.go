package node

import (
	"crypto/sha1"
	"fmt"
	pid "my-kad-dht/core/id"
)

func (n *Node) DumpStorage() {
	fmt.Printf("Storage of node: %s", n.ID())
	n.KVStorage.Print()
}

func hashKey(key string) pid.PeerID {
	h := sha1.Sum([]byte(key))
	return pid.PeerID(fmt.Sprintf("%x", h))
}
