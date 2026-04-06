package network

import (
	"crypto/sha1"
	"fmt"
	pid "my-kad-dht/internal/id"
)

func hashKey(key string) pid.PeerID {
	h := sha1.Sum([]byte(key))
	return pid.PeerID(fmt.Sprintf("%x", h))
}