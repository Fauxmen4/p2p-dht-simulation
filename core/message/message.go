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
	StoreCacheType
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
	
	Success bool // equals true in case of successful PING/STORE
}

type FindNodeBody struct{ TargetID string }

type FindValueBody struct {
	TargetID string
	Bitmap   uint64 // Shades: bitmask of colors the sender already has in its palette
}

type StoreBody struct{ Key, Value string }
type StoreCacheBody struct{ Key, Value string }

type FindNodeResponse struct {
	Nearest []rt.PeerInfo
}

type FindValueResponse struct {
	Value   string
	Nearest []rt.PeerInfo

	// Shades fields — zero values are safe for non-Shades nodes
	IsNeeded   bool          // true if this node wants to cache the value
	ColorNodes []rt.PeerInfo // peers to merge into the requester's palette
}
