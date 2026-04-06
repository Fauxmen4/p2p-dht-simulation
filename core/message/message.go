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
