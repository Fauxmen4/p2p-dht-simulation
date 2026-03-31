package metrics

type search struct {
	total   int
	success int
}

type hopInfo struct {
	key     string
	hops    int
	success bool
}

// Storage represents some metrics about node work
type Storage struct {
	// shows load of the node
	handledRPCs int

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

func (s *Storage) NewRPC() {
	s.handledRPCs += 1
}

func (s *Storage) CountRPCs() int {
	return s.handledRPCs
}

func (s *Storage) NewSearch(key string, hops int, success bool) {
	// add search info
	s.search.total += 1
	if success {
		s.search.success += 1
	}

	// add hop info
	s.hopHistory = append(s.hopHistory, hopInfo{
		key:     key,
		hops:    hops,
		success: success,
	})
}

func (s *Storage) SearchInfo() []hopInfo {
	return s.hopHistory
}

func (s *Storage) ReturnHops() []int {
	hops_count := []int{}
	for _, hops := range s.hopHistory {
		hops_count = append(hops_count, hops.hops)
	}

	return hops_count
}
