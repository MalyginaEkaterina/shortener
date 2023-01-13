package storage

import (
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

type Storage interface {
	AddURL(url string) (int, error)
	GetURL(id string) (string, error)
}
