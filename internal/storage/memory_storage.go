package storage

import (
	"context"
	"github.com/MalyginaEkaterina/shortener/internal"
	"strconv"
	"sync"
	"sync/atomic"
)

// URL represents a URL stored in MemoryStorage.
type URL struct {
	url       string
	userID    int32
	isDeleted bool
}

var _ Storage = (*MemoryStorage)(nil)

// MemoryStorage represents an in-memory storage implementation of the Storage interface.
type MemoryStorage struct {
	urls      []URL
	userCount atomic.Int32
	UserUrls  map[int32][]int32
	UrlsID    map[string]int32
	mutex     sync.RWMutex
}

// NewMemoryStorage creates new *MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{UserUrls: make(map[int32][]int32), UrlsID: make(map[string]int32)}
}

// AddUser adds a new user to MemoryStorage and returns its id.
func (s *MemoryStorage) AddUser(_ context.Context) (int, error) {
	return int(s.userCount.Add(1)), nil
}

// AddURL adds a new URL to MemoryStorage and returns its id, or an error if the URL already exists.
func (s *MemoryStorage) AddURL(_ context.Context, url string, userID int) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.UrlsID[url]
	if ok {
		return 0, ErrAlreadyExists
	}
	s.urls = append(s.urls, URL{url: url, userID: int32(userID), isDeleted: false})
	urlID := len(s.urls) - 1
	s.UrlsID[url] = int32(urlID)
	s.UserUrls[int32(userID)] = append(s.UserUrls[int32(userID)], int32(urlID))
	return urlID, nil
}

// GetURL returns the original URL corresponding to the given id, or an error if not found or deleted.
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

// GetURLID returns the id by the given original URL.
func (s *MemoryStorage) GetURLID(_ context.Context, url string) (int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	id := s.UrlsID[url]
	return int(id), nil
}

// GetUserUrls returns a map with ids and their original URLs for all URLs for the user.
func (s *MemoryStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	urlIDs, ok := s.UserUrls[int32(userID)]
	if !ok {
		return nil, ErrNotFound
	}
	res := make(map[int]string)
	for _, urlID := range urlIDs {
		res[int(urlID)] = s.urls[urlID].url
	}
	return res, nil
}

// AddBatch adds a list of new URLs to MemoryStorage and returns their corresponding correlation IDs and URL IDs.
func (s *MemoryStorage) AddBatch(_ context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var res []internal.CorrIDUrlID
	for _, v := range urls {
		corrIDUrlID := internal.CorrIDUrlID{CorrID: v.CorrID}
		s.urls = append(s.urls, URL{url: v.OriginalURL, userID: int32(userID), isDeleted: false})
		corrIDUrlID.URLID = len(s.urls) - 1
		s.UserUrls[int32(userID)] = append(s.UserUrls[int32(userID)], int32(corrIDUrlID.URLID))
		res = append(res, corrIDUrlID)
	}
	return res, nil
}

// DeleteBatch marks a list of URLs as deleted in MemoryStorage.
func (s *MemoryStorage) DeleteBatch(_ context.Context, ids []internal.IDToDelete) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, v := range ids {
		if v.ID >= 0 || v.ID < len(s.urls) {
			url := s.urls[v.ID]
			if url.userID == int32(v.UserID) {
				url.isDeleted = true
				s.urls[v.ID] = url
			}
		}
	}
	return nil
}

// Close does nothing.
func (s *MemoryStorage) Close() {
}
