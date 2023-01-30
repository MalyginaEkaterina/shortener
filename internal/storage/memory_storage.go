package storage

import "strconv"

type MemoryStorage struct {
	Urls      []string
	UserCount int
	UserUrls  map[int][]int
}

var _ Storage = (*MemoryStorage)(nil)

func (s *MemoryStorage) AddUser() (int, error) {
	s.UserCount++
	return s.UserCount, nil
}

func (s *MemoryStorage) AddURL(url string, userID int) (int, error) {
	s.Urls = append(s.Urls, url)
	urlID := len(s.Urls) - 1
	s.UserUrls[userID] = append(s.UserUrls[userID], urlID)
	return urlID, nil
}

func (s *MemoryStorage) GetURL(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}

func (s *MemoryStorage) GetUserUrls(userID int) (map[int]string, error) {
	urlIDs, ok := s.UserUrls[userID]
	if !ok {
		return nil, ErrNotFound
	}
	res := make(map[int]string)
	for _, urlID := range urlIDs {
		res[urlID] = s.Urls[urlID]
	}
	return res, nil
}

func (s *MemoryStorage) Close() {
}
