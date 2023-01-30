package storage

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type CachedFileStorage struct {
	File      *os.File
	Urls      []string
	UserCount int
	UserUrls  map[int][]int
}

var _ Storage = (*CachedFileStorage)(nil)

func NewCachedFileStorage(filename string) (*CachedFileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	var urls []string
	var userCount int
	userUrls := make(map[int][]int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := string(scanner.Bytes())
		d := strings.Split(str, " ")
		userID, err := strconv.Atoi(d[0])
		if err != nil {
			return nil, err
		}
		if userCount < userID {
			userCount = userID
		}
		urls = append(urls, d[1])
		urlID := len(urls) - 1
		userUrls[userID] = append(userUrls[userID], urlID)
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return &CachedFileStorage{File: file, Urls: urls, UserCount: userCount, UserUrls: userUrls}, nil
}

func (s *CachedFileStorage) Close() {
	s.File.Close()
}

func (s *CachedFileStorage) AddUser() (int, error) {
	s.UserCount++
	return s.UserCount, nil
}

func (s *CachedFileStorage) AddURL(url string, userID int) (int, error) {
	data := []byte(strconv.Itoa(userID) + " " + url + "\n")
	_, err := s.File.Write(data)
	if err != nil {
		return 0, err
	}
	s.Urls = append(s.Urls, url)
	urlID := len(s.Urls) - 1
	s.UserUrls[userID] = append(s.UserUrls[userID], urlID)
	return urlID, nil
}

func (s *CachedFileStorage) GetURL(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}

func (s *CachedFileStorage) GetUserUrls(userID int) (map[int]string, error) {
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
