package metrics

import "time"

// success rate
type search struct {
	total   int
	success int
}

// info about one exact lookup
type hopInfo struct {
	key      string
	hops     int
	success  bool
	duration time.Duration
}

// Storage represents some metrics about node work
type Storage struct {
	// shows load of the node
	handledRPCs int
	sentRPCs    int

	search search

	hopHistory []hopInfo
}

func NewStorage() *Storage {
	s := &Storage{
		handledRPCs: 0,
		search: search{
			total:   0,
			success: 0,
		},
		hopHistory: make([]hopInfo, 0),
	}

	return s
}

// isSent = false means that its inbound RPC
func (s *Storage) NewRPC(isSent bool) {
	if isSent {
		s.sentRPCs++
	} else {
		s.handledRPCs++
	}
}

func (s *Storage) SentRPCs() int {
	return s.sentRPCs
}

func (s *Storage) HandledRPCs() int {
	return s.handledRPCs
}

func (s *Storage) NewSearch(key string, hops int, success bool, duration time.Duration) {
	// add search info
	s.search.total += 1
	if success {
		s.search.success += 1
	}

	// add hop info
	s.hopHistory = append(s.hopHistory, hopInfo{
		key:      key,
		hops:     hops,
		success:  success,
		duration: duration,
	})
}

func (s *Storage) SearchHistory() []hopInfo {
	return s.hopHistory
}

func (s *Storage) SuccessHopCount() ([]int, []float64) {
	hopCounts := []int{}
	durations := []float64{}
	for _, hopInfo := range s.hopHistory {
		if hopInfo.success {
			hopCounts = append(hopCounts, hopInfo.hops)
			durations = append(durations, hopInfo.duration.Seconds())
		}
	}
	return hopCounts, durations
}

func (s *Storage) CountKeyLookups() int {
	return s.search.total
}
