package storage

import "strconv"

type MemoryStorage struct {
	Urls []string
}

var _ Storage = (*MemoryStorage)(nil)

func (s *MemoryStorage) AddURL(url string) (int, error) {
	s.Urls = append(s.Urls, url)
	return len(s.Urls) - 1, nil
}

func (s *MemoryStorage) GetURL(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}

func (s *MemoryStorage) Close() {
}
