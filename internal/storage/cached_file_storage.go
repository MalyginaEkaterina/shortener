package storage

import (
	"bufio"
	"os"
	"strconv"
)

type CachedFileStorage struct {
	Filename string
	Urls     []string
}

var _ Storage = (*CachedFileStorage)(nil)

func NewCachedFileStorage(filename string) (*CachedFileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := scanner.Bytes()
		urls = append(urls, string(str))
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return &CachedFileStorage{Filename: filename, Urls: urls}, nil
}

func (s *CachedFileStorage) AddURL(url string) (int, error) {
	file, err := os.OpenFile(s.Filename, os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return -1, err
	}
	defer file.Close()
	data := []byte(url)
	data = append(data, '\n')
	_, err = file.Write(data)
	if err != nil {
		return -1, err
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
