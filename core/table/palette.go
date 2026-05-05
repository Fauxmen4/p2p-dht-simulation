// Palette for implementing Shades caching & routing strategy
package routingtable

import (
	pid "my-kad-dht/core/id"
	bs "my-kad-dht/pkg/bitset"
	"sync"
)

type Palette struct {
	mu      sync.RWMutex
	ownerID pid.PeerID
	buckets map[uint8][]PeerInfo // color -> list of contacts
	index   map[pid.PeerID]uint8 // nodeID -> color

	colors  uint8     // number of possible colors
	bitmask bs.BitSet // bitmask has 0 for empty colors
}

func NewPalette(ownerID pid.PeerID, colors uint8) *Palette {
	return &Palette{
		ownerID: ownerID,
		buckets: make(map[uint8][]PeerInfo),
		index:   make(map[pid.PeerID]uint8),
		colors:  colors,
		bitmask: bs.BitSet{},
	}
}

// Add adds contact to palette and returns true in case it was added.
func (p *Palette) Add(pi PeerInfo) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.index[pi.Id]; ok {
		return false
	}

	if _, ok := p.buckets[pi.Color]; !ok {
		p.bitmask.Set(pi.Color)
		p.buckets[pi.Color] = []PeerInfo{}
	}
	p.buckets[pi.Color] = append(p.buckets[pi.Color], pi)
	p.index[pi.Id] = pi.Color

	return true
}

func (p *Palette) Remove(id pid.PeerID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.index[id]; !ok {
		return false
	}

	color := pid.ColorId(pid.ConvertPeerID(id), p.colors)

	bucket := p.buckets[color]
	for i := range bucket {
		if bucket[i].Id == id {
			bucket[i] = bucket[len(bucket)-1]
			p.buckets[color] = bucket[:len(bucket)-1]
			if len(p.buckets[color]) == 0 {
				delete(p.buckets, color)
				p.bitmask.Unset(color)
			}
			break
		}
	}

	delete(p.index, id)

	return true
}

func (p *Palette) Bitmask() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.bitmask.Bits()
}

// ClosestToKey returns the closest node of the same color as key.
// Returns false if the palette has no nodes of that color.
func (p *Palette) ClosestToKey(key pid.ID) (PeerInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	color := pid.ColorId(key, p.colors)
	return ClosestPeer(p.buckets[color], key)
}

// GetNodesByBitmask returns one contact per color that has a 1-bit in bitMask,
// skipping colors the palette has no nodes for.
func (p *Palette) GetNodesByBitmask(bitMask uint64, k int) []PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]PeerInfo, 0)

	for i := uint8(0); i < p.colors; i++ {
		if (bitMask & (1 << i)) != 0 { // if this color needs to be included in result
			if _, ok := p.buckets[i]; ok { // if palette contains nodes of this color
				// add random (first) contact
				result = append(result, p.buckets[i][0])
			}
		}
	}

	return result
}
