package storage

import (
	"context"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrDeleted       = errors.New("was deleted")
)

type Storage interface {
	AddUser(ctx context.Context) (int, error)
	AddURL(ctx context.Context, url string, userID int) (int, error)
	GetURLID(ctx context.Context, url string) (int, error)
	GetURL(ctx context.Context, id string) (string, error)
	GetUserUrls(ctx context.Context, userID int) (map[int]string, error)
	AddBatch(ctx context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error)
	DeleteBatch(ctx context.Context, ids []internal.IDToDelete) error
	Close()
}
