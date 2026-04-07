package node

import (
	"my-kad-dht/core/addr"
	msg "my-kad-dht/core/message"
)

type Transport interface {
	// SendAsync delivers a message in a fire-and-forget manner.
	SendAsync(to addr.Addr, m *msg.Message)
}
