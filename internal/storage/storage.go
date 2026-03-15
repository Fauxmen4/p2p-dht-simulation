package storage

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
