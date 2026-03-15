package addr

import (
	"fmt"
	"math/rand/v2"

	"github.com/brianvoe/gofakeit/v7"
)

// Addr is a "host:port" string which imitates peer real network address
type Addr string

func GenerateAddr() Addr {
	return Addr(fmt.Sprintf(
		"%s:%d",
		gofakeit.IPv4Address(),
		rand.IntN(65535-1024)+1024,
	))
}
