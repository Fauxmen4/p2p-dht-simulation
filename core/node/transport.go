package node

import (
	"my-kad-dht/core/addr"
	msg "my-kad-dht/core/message"
)

type Transport interface {
	SendAsync(to addr.Addr, m *msg.Message)
}
