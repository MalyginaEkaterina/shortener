package storage

import (
	"context"
	"github.com/MalyginaEkaterina/shortener/internal"
	"strconv"
	"sync"
	"sync/atomic"
)

type URL struct {
	url       string
	userID    int
	isDeleted bool
}

type MemoryStorage struct {
	urls      []URL
	userCount atomic.Int32
	UserUrls  map[int][]int
	UrlsID    map[string]int
	mutex     sync.RWMutex
}

var _ Storage = (*MemoryStorage)(nil)

func (s *MemoryStorage) AddUser(_ context.Context) (int, error) {
	return int(s.userCount.Add(1)), nil
}

func (s *MemoryStorage) AddURL(_ context.Context, url string, userID int) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.UrlsID[url]
	if ok {
		return 0, ErrAlreadyExists
	}
	s.urls = append(s.urls, URL{url: url, userID: userID, isDeleted: false})
	urlID := len(s.urls) - 1
	s.UrlsID[url] = urlID
	s.UserUrls[userID] = append(s.UserUrls[userID], urlID)
	return urlID, nil
}

func (s *MemoryStorage) GetURL(_ context.Context, idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.urls) {
		return "", ErrNotFound
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	url := s.urls[id]
	if url.isDeleted {
		return "", ErrDeleted
	}
	return url.url, err
}

func (s *MemoryStorage) GetURLID(_ context.Context, url string) (int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	id := s.UrlsID[url]
	return id, nil
}

func (s *MemoryStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	urlIDs, ok := s.UserUrls[userID]
	if !ok {
		return nil, ErrNotFound
	}
	res := make(map[int]string)
	for _, urlID := range urlIDs {
		res[urlID] = s.urls[urlID].url
	}
	return res, nil
}

func (s *MemoryStorage) AddBatch(_ context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var res []internal.CorrIDUrlID
	for _, v := range urls {
		corrIDUrlID := internal.CorrIDUrlID{CorrID: v.CorrID}
		s.urls = append(s.urls, URL{url: v.OriginalURL, userID: userID, isDeleted: false})
		corrIDUrlID.URLID = len(s.urls) - 1
		s.UserUrls[userID] = append(s.UserUrls[userID], corrIDUrlID.URLID)
		res = append(res, corrIDUrlID)
	}
	return res, nil
}

func (s *MemoryStorage) DeleteBatch(_ context.Context, ids []internal.IDToDelete) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, v := range ids {
		if v.ID >= 0 || v.ID < len(s.urls) {
			url := s.urls[v.ID]
			if url.userID == v.UserID {
				url.isDeleted = true
				s.urls[v.ID] = url
			}
		}
	}
	return nil
}

func (s *MemoryStorage) Close() {
}
