package config

import (
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
)

type NodeSpec struct {
	ID           pid.PeerID
	Addr         addr.Addr
	BootstrapVia []pid.PeerID
}
