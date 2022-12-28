package storage

import "strconv"

type Storage interface {
	AddURL(url string) (int, error)
	ValidID(id string) (bool, error)
	GetURL(id string) (string, error)
}

type MemoryStorage struct {
	Urls []string
}

var _ Storage = (*MemoryStorage)(nil)

func (s *MemoryStorage) AddURL(url string) (int, error) {
	s.Urls = append(s.Urls, url)
	return len(s.Urls) - 1, nil
}

func (s *MemoryStorage) ValidID(idStr string) (bool, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return false, nil
	}
	return id >= 0 && id < len(s.Urls), nil
}

func (s *MemoryStorage) GetURL(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	return s.Urls[id], err
}
