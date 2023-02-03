package storage

import (
	"context"
	"github.com/MalyginaEkaterina/shortener/internal"
	"strconv"
)

type MemoryStorage struct {
	Urls      []string
	UserCount int
	UserUrls  map[int][]int
}

var _ Storage = (*MemoryStorage)(nil)

func (s *MemoryStorage) AddUser(_ context.Context) (int, error) {
	s.UserCount++
	return s.UserCount, nil
}

func (s *MemoryStorage) AddURL(_ context.Context, url string, userID int) (int, error) {
	s.Urls = append(s.Urls, url)
	urlID := len(s.Urls) - 1
	s.UserUrls[userID] = append(s.UserUrls[userID], urlID)
	return urlID, nil
}

func (s *MemoryStorage) GetURL(_ context.Context, idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}

func (s *MemoryStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
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

func (s *MemoryStorage) AddBatch(_ context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	var res []internal.CorrIDUrlID
	for _, v := range urls {
		corrIDUrlID := internal.CorrIDUrlID{CorrID: v.CorrID}
		s.Urls = append(s.Urls, v.OriginalURL)
		corrIDUrlID.URLID = len(s.Urls) - 1
		s.UserUrls[userID] = append(s.UserUrls[userID], corrIDUrlID.URLID)
		res = append(res, corrIDUrlID)
	}
	return res, nil
}

func (s *MemoryStorage) Close() {
}
