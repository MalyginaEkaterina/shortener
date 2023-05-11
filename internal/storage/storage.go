package storage

import (
	"context"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
)

// Storage errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrDeleted       = errors.New("was deleted")
)

// Storage.
type Storage interface {
	// AddUser creates a new user into storage and returns its id.
	AddUser(ctx context.Context) (int, error)
	// AddURL saves URL for the user into storage and returns its id.
	AddURL(ctx context.Context, url string, userID int) (int, error)
	// GetURLID returns id of URL.
	GetURLID(ctx context.Context, url string) (int, error)
	// GetURL returns URL by its id.
	GetURL(ctx context.Context, id string) (string, error)
	// GetUserUrls returns map of id and URL with all URLs for the user.
	GetUserUrls(ctx context.Context, userID int) (map[int]string, error)
	// AddBatch saves the batch of URLs for the user. Returns array of url IDs and its CorrID.
	AddBatch(ctx context.Context, urls []internal.CorrIDOriginalURL, userID int) ([]internal.CorrIDUrlID, error)
	// DeleteBatch marks url IDs from the list as deleted in storage.
	DeleteBatch(ctx context.Context, ids []internal.IDToDelete) error
	// GetStat returns count of shortened URL and count of User
	GetStat(ctx context.Context) (urls, users int, err error)
	// Close closes resources.
	Close()
}
