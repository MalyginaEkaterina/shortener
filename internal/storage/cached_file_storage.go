package storage

import (
	"bufio"
	"os"
	"strconv"
)

type CachedFileStorage struct {
	File *os.File
	Urls []string
}

var _ Storage = (*CachedFileStorage)(nil)

func NewCachedFileStorage(filename string) (*CachedFileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := scanner.Bytes()
		urls = append(urls, string(str))
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return &CachedFileStorage{File: file, Urls: urls}, nil
}

func (s *CachedFileStorage) Close() {
	s.File.Close()
}

func (s *CachedFileStorage) AddURL(url string) (int, error) {
	data := []byte(url + "\n")
	_, err := s.File.Write(data)
	if err != nil {
		return 0, err
	}
	s.Urls = append(s.Urls, url)
	return len(s.Urls) - 1, nil
}

func (s *CachedFileStorage) GetURL(idStr string) (string, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.Urls) {
		return "", ErrNotFound
	}
	return s.Urls[id], err
}
