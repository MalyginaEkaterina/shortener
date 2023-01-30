package storage

import (
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

type Storage interface {
	AddUser() (int, error)
	AddURL(url string, userID int) (int, error)
	GetURL(id string) (string, error)
	GetUserUrls(userID int) (map[int]string, error)
	Close()
}
