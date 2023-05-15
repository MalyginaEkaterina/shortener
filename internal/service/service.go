package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
)

// Service is service between Storage and handlers.
type Service interface {
	// AddURL saves URL into storage. If this URL already exists then gets its ID.
	// Returns ID of shortened URL and a flag if the URL existed.
	AddURL(ctx context.Context, url string, userID int) (int, bool, error)
	GetUserID(sign string) (int, error)
	GetUserIDOrCreate(ctx context.Context, sign string) (int, string, error)
}

var _ Service = (*URLService)(nil)

// ErrSignNotValid is sign error
var (
	ErrSignNotValid = errors.New("sign is not valid")
)

// URLService contains storage.
type URLService struct {
	Store  storage.Storage
	Signer Signer
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

// GetUserID returns userID from the sign
func (u URLService) GetUserID(sign string) (int, error) {
	userID, authOK, err := u.Signer.CheckSign(sign)
	if err != nil {
		log.Println("Error while checking of sign", err)
		return 0, err
	}
	if !authOK {
		return 0, ErrSignNotValid
	}
	return userID, nil
}

// GetUserIDOrCreate checks user`s sign and creates a new user if required
func (u URLService) GetUserIDOrCreate(ctx context.Context, sign string) (int, string, error) {
	if sign == "" {
		return u.createUser(ctx)
	}
	userID, authOK, err := u.Signer.CheckSign(sign)
	if err != nil {
		log.Println("Error while checking of sign", err)
		return 0, "", err
	}
	if !authOK {
		return u.createUser(ctx)
	}
	return userID, sign, nil
}

// CreateUser creates user and returns its id and sign
func (u URLService) createUser(ctx context.Context) (int, string, error) {
	userID, err := u.Store.AddUser(ctx)
	if err != nil {
		log.Println("Error while adding user", err)
		return 0, "", err
	}
	signValue, err := u.Signer.CreateSign(userID)
	if err != nil {
		log.Println("Error while creating of sign", err)
		return 0, "", err
	}
	return userID, signValue, err
}
