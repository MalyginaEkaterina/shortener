package storage

import (
	"bufio"
	"context"
	"github.com/MalyginaEkaterina/shortener/internal"
	"os"
	"strconv"
	"strings"
)

type CachedFileStorage struct {
	File      *os.File
	Urls      []string
	UserCount int
	UserUrls  map[int][]int
	UrlsID    map[string]int
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
	urlsID := make(map[string]int)
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
		urlsID[d[1]] = urlID
		userUrls[userID] = append(userUrls[userID], urlID)
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return &CachedFileStorage{File: file, Urls: urls, UserCount: userCount, UserUrls: userUrls, UrlsID: urlsID}, nil
}

func (s *CachedFileStorage) Close() {
	s.File.Close()
}

func (s *CachedFileStorage) AddUser(_ context.Context) (int, error) {
	s.UserCount++
	return s.UserCount, nil
}

func (s *CachedFileStorage) AddURL(_ context.Context, url string, userID int) (int, error) {
	_, ok := s.UrlsID[url]
	if ok {
		return 0, ErrAlreadyExists
	}
	data := []byte(strconv.Itoa(userID) + " " + url + "\n")
	_, err := s.File.Write(data)
	if err != nil {
		return 0, err
	}
	s.Urls = append(s.Urls, url)
	urlID := len(s.Urls) - 1
	s.UrlsID[url] = urlID
	s.UserUrls[userID] = append(s.UserUrls[userID], urlID)
	return urlID, nil
}

func (s *CachedFileStorage) GetURL(_ context.Context, idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}

func (s *CachedFileStorage) GetURLID(_ context.Context, url string) (int, error) {
	return s.UrlsID[url], nil
}

func (s *CachedFileStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
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

func (s *CachedFileStorage) AddBatch(_ context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error) {
	var res []internal.CorrIDUrlID
	for _, v := range urls {
		data := []byte(strconv.Itoa(userID) + " " + v.OriginalURL + "\n")
		_, err := s.File.Write(data)
		if err == nil {
			corrIDUrlID := internal.CorrIDUrlID{CorrID: v.CorrID}
			s.Urls = append(s.Urls, v.OriginalURL)
			corrIDUrlID.URLID = len(s.Urls) - 1
			s.UserUrls[userID] = append(s.UserUrls[userID], corrIDUrlID.URLID)
			res = append(res, corrIDUrlID)
		}
	}
	return res, nil
}
