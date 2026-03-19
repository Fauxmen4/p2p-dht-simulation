package metrics

// Storage represents some metrics about node work
type Storage struct {
    // shows load of the node
    handledRPCs int

}

func NewStorage() *Storage {
    return &Storage{
        handledRPCs: 0,
    }
}

func (s *Storage) NewRPC() {
    s.handledRPCs += 1
}

func (s *Storage) CountRPCs() int {
    return s.handledRPCs
}