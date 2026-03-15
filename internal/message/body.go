package message

import rt "my-kad-dht/internal/table"

type MsgBody interface {
	isBody() bool
}

var (
	_ MsgBody = (*Body)(nil)
)

type Body struct {
	// request fields
	ID         string
	Key        string
	InputValue string

	// response fields
	NearestNodes []rt.PeerInfo
	OutputValue  string
}

func (b *Body) isBody() bool {
	return true
}
