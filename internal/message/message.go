package message

import (
	"my-kad-dht/internal/addr"
	pid "my-kad-dht/internal/id"
)

type MsgID string

type MsgType int

const (
	PingType MsgType = iota + 1
	StoreType
	FindNodeType
	FindValueType
)

type Message interface {
	Receiver() addr.Addr
}

var (
	_ Message = (*Request)(nil)
	_ Message = (*Response)(nil)
)

type Request struct {
	ID   MsgID
	Type MsgType

	Body Body

	// node addrs
	To     addr.Addr
	From   addr.Addr
	FromID pid.PeerID
}

func (r *Request) Receiver() addr.Addr {
	return r.To
}

type Response struct {
	ID   MsgID
	Type MsgType

	Body Body

	// node addrs
	To   addr.Addr
	From addr.Addr
	FromID pid.PeerID
}

func (r *Response) Receiver() addr.Addr {
	return r.To
}
