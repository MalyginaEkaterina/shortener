package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
)

type Service interface {
	AddURL(ctx context.Context, url string, userID int) (int, bool, error)
}

type URLService struct {
	Store storage.Storage
}

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

var _ Service = (*URLService)(nil)
