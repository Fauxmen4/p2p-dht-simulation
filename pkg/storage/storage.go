package storage

import "fmt"

// Storage represents simple in-memory storage
type Storage struct {
	data map[string]string
}

func New() *Storage {
	return &Storage{
		data: make(map[string]string, 0),
	}
}

func (s *Storage) Set(key, value string) {
	s.data[key] = value
}

func (s *Storage) Get(key string) (string, bool) {
	value, ok := s.data[key]
	return value, ok
}

func (s *Storage) Delete(key string) {
	delete(s.data, key)
}

// Print prints all (key, value) pairs from storage
func (s *Storage) Print() {
	fmt.Printf("Number of records: %d\n", len(s.data))
	for key, value := range s.data {
		fmt.Printf("\t- %s:%s\n", key, value)
	}
}
