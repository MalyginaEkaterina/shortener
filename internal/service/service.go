package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
)

// Service is service between Storage and handlers.
type Service interface {
	// AddURL saves URL into storage. If this URL already exists then gets its ID.
	// Returns ID of shortened URL and a flag if the URL existed.
	AddURL(ctx context.Context, url string, userID int) (int, bool, error)
}

var _ Service = (*URLService)(nil)

// URLService contains storage.
type URLService struct {
	Store storage.Storage
}

// AddURL saves URL into storage. If this URL already exists then gets its ID.
// Returns ID of shortened URL and a flag if the URL existed.
func (u URLService) AddURL(ctx context.Context, url string, userID int) (int, bool, error) {
	ind, err := u.Store.AddURL(ctx, url, userID)
	if errors.Is(err, storage.ErrAlreadyExists) {
		ind, err = u.Store.GetURLID(ctx, url)
		if err != nil {
			return 0, false, fmt.Errorf(`error while getting url id: %w`, err)
		}
		return ind, true, nil
	} else if err != nil {
		return 0, false, fmt.Errorf(`error while adding url: %w`, err)
	}
	return ind, false, nil
}
