package message

import (
	"my-kad-dht/core/addr"
	pid "my-kad-dht/core/id"
	rt "my-kad-dht/core/table"
)

type MsgID string

type MsgType int

const (
	PingType MsgType = iota + 1
	StoreType
	FindNodeType
	FindValueType
)

type Message struct {
	ID     MsgID
	Type   MsgType
	Body   any
	To     addr.Addr
	From   addr.Addr
	FromID pid.PeerID
	// IsResponse == true if it's already handled message
	// and should be passed to client operation function
	IsResponse bool
}

type FindNodeBody struct{ TargetID string }
type FindValueBody struct{ TargetID string }
type StoreBody struct{ Key, Value string }

type FindNodeResponse struct {
	Nearest []rt.PeerInfo
}

type FindValueResponse struct {
	Value   string
	Nearest []rt.PeerInfo
}
