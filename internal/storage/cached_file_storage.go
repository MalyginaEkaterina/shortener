package storage

import (
	"bufio"
	"context"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal"
	"os"
	"strconv"
	"strings"
	"sync"
)

var _ Storage = (*CachedFileStorage)(nil)

// CachedFileStorage uses file for storage and cache in memory.
type CachedFileStorage struct {
	file     *os.File
	filename string
	urlCount int

	fileMutex sync.Mutex

	urls       map[int]URL
	userCount  int
	userUrls   map[int][]int
	urlsID     map[string]int
	cacheMutex sync.RWMutex
}

// NewCachedFileStorage creates CachedFileStorage and fills memory storage from the file with name=filename.
func NewCachedFileStorage(filename string) (*CachedFileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	urls := make(map[int]URL)
	var userCount int
	var urlCount int
	userUrls := make(map[int][]int)
	urlsID := make(map[string]int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := string(scanner.Bytes())
		d := strings.Split(str, " ")
		var id int
		id, err = strconv.Atoi(d[0])
		if err != nil {
			return nil, err
		}
		if urlCount < id {
			urlCount = id
		}
		var userID int
		userID, err = strconv.Atoi(d[1])
		if err != nil {
			return nil, err
		}
		if userCount < userID {
			userCount = userID
		}
		var isDeleted bool
		isDeleted, err = strconv.ParseBool(d[3])
		if err != nil {
			return nil, err
		}
		url := URL{userID: int32(userID), url: d[2], isDeleted: isDeleted}
		urls[id] = url
		urlsID[d[1]] = id
		userUrls[userID] = append(userUrls[userID], id)
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return &CachedFileStorage{
		file:      file,
		filename:  filename,
		urls:      urls,
		userCount: userCount,
		userUrls:  userUrls,
		urlsID:    urlsID,
		urlCount:  urlCount,
	}, nil
}

// Close closes the file.
func (s *CachedFileStorage) Close() {
	s.file.Close()
}

// AddUser adds new user.
func (s *CachedFileStorage) AddUser(_ context.Context) (int, error) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.userCount++
	return s.userCount, nil
}

// AddURL saves URL into file and after that saves it into cache. Returns ErrAlreadyExists if URL has been added already.
func (s *CachedFileStorage) AddURL(_ context.Context, url string, userID int) (int, error) {
	s.fileMutex.Lock()
	defer s.fileMutex.Unlock()
	if _, ok := s.urlsID[url]; ok {
		return 0, ErrAlreadyExists
	}

	s.urlCount++
	id := s.urlCount
	_, err := fmt.Fprintf(s.file, "%d %d %s %v\n", id, userID, url, false)
	if err != nil {
		return 0, err
	}

	s.addToCache(userID, url, id)
	return id, nil
}

// GetURL returns URL by ID from cache. Returns ErrNotFound if URL does not exist and ErrDeleted if URL is marked as deleted.
func (s *CachedFileStorage) GetURL(_ context.Context, idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return "", ErrNotFound
	}
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	url, ok := s.urls[id]
	if !ok {
		return "", ErrNotFound
	}
	if url.isDeleted {
		return "", ErrDeleted
	}
	return url.url, nil
}

// GetURLID returns url ID from cache.
func (s *CachedFileStorage) GetURLID(_ context.Context, url string) (int, error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return s.urlsID[url], nil
}

// GetUserUrls returns map with ids and original urls for all user's urls from cache.
func (s *CachedFileStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	urlIDs, ok := s.userUrls[userID]
	if !ok {
		return nil, ErrNotFound
	}
	res := make(map[int]string)
	for _, urlID := range urlIDs {
		res[urlID] = s.urls[urlID].url
	}
	return res, nil
}

// AddBatch saves list of urls into file and into cache.
func (s *CachedFileStorage) AddBatch(_ context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	s.fileMutex.Lock()
	defer s.fileMutex.Unlock()

	var res []internal.CorrIDUrlID
	for _, v := range urls {
		if _, ok := s.urlsID[v.OriginalURL]; ok {
			continue
		}
		s.urlCount++
		id := s.urlCount
		_, err := fmt.Fprintf(s.file, "%d %d %s %v\n", id, userID, v.OriginalURL, false)
		if err == nil {
			res = append(res, internal.CorrIDUrlID{CorrID: v.CorrID, URLID: id})
			s.addToCache(userID, v.OriginalURL, id)
		}
	}
	return res, nil
}

// DeleteBatch rewrites file marking URLs from the list as deleted in file and in cache.
func (s *CachedFileStorage) DeleteBatch(_ context.Context, ids []internal.IDToDelete) error {
	// TODO: Change int to specific types.
	idsMap := make(map[int]int)
	for _, id := range ids {
		idsMap[id.ID] = id.UserID
	}

	s.fileMutex.Lock()
	defer s.fileMutex.Unlock()
	tmpPath := "tmp_" + s.filename
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	for id, url := range s.urls {
		userID, deleted := idsMap[id]
		if deleted {
			if int32(userID) != url.userID {
				deleted = false
			}
		}
		if deleted {
			s.setDeletedInCache(id)
		}
		_, err = fmt.Fprintf(tmpFile, "%d %d %s %v\n", id, url.userID, url.url, deleted)
		if err != nil {
			return err
		}
	}
	err = s.file.Close()
	if err != nil {
		return err
	}
	err = tmpFile.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tmpPath, s.filename)
	if err != nil {
		return err
	}
	s.file, err = os.OpenFile(s.filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	return nil
}

func (s *CachedFileStorage) setDeletedInCache(id int) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	url := s.urls[id]
	url.isDeleted = true
	s.urls[id] = url
}

func (s *CachedFileStorage) addToCache(userID int, url string, id int) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.urls[id] = URL{userID: int32(userID), url: url, isDeleted: false}
	s.urlsID[url] = id
	s.userUrls[userID] = append(s.userUrls[userID], id)
}

// GetStat returns count of shortened URL and count of User
func (s *CachedFileStorage) GetStat(_ context.Context) (urls, users int, err error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return len(s.urls), s.userCount, nil
}
