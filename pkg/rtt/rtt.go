package rtt

import (
	"math"
	"time"
)

type Coord struct {
	X      float64
	Y      float64
	Height float64
}

// OneWayDelay calculates time for sending a single packet.
func OneWayDelay(a, b Coord) time.Duration {
	return rtt(a, b) / 2
}

func rtt(a, b Coord) time.Duration {
	dx, dy := a.X-b.X, a.Y-b.Y
	rttMs := math.Sqrt(dx*dx+dy*dy) + a.Height + b.Height
	return time.Duration(rttMs * float64(time.Millisecond))
}
