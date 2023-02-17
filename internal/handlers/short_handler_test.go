package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/service"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUrlById(t *testing.T) {
	type want struct {
		statusCode int
		url        string
	}
	tests := []struct {
		name  string
		path  string
		store storage.Storage
		want  want
	}{
		{
			name:  "Positive test with correct url id",
			path:  "/1",
			store: &mockStorage{getURL: "http://test.ru"},
			want:  want{307, "http://test.ru"},
		},
		{
			name:  "Negative test with incorrect url id #1",
			path:  "/1",
			store: &mockStorage{getURLErr: storage.ErrNotFound},
			want:  want{400, ""},
		},
		{
			name:  "Negative test with getURLError",
			path:  "/1",
			store: &mockStorage{getURLErr: errors.New("negative test with getURLError")},
			want:  want{500, ""},
		},
		{
			name:  "Negative test with empty url id",
			path:  "/",
			store: &mockStorage{},
			want:  want{400, ""},
		},
		{
			name:  "Negative test with empty url id #2",
			path:  "",
			store: &mockStorage{},
			want:  want{400, ""},
		},
	}
	cfg := internal.Config{Address: ":8080", BaseURL: "http://localhost:8080"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store, cfg, Signer{SecretKey: []byte("secret again")}, service.URLService{Store: tt.store})
			ts := httptest.NewServer(r)
			defer ts.Close()

			request := httptest.NewRequest(http.MethodGet, ts.URL+tt.path, nil)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, request)

			assert.Equal(t, tt.want.statusCode, resp.Code)
			if tt.want.url != "" {
				assert.Equal(t, tt.want.url, resp.Header().Get("Location"))
			}
		})
	}

}

func TestShortUrl(t *testing.T) {
	type want struct {
		statusCode int
		url        string
	}
	tests := []struct {
		name    string
		request string
		store   storage.Storage
		want    want
	}{
		{
			name:    "Positive test",
			request: "http://test.ru",
			store:   &mockStorage{addURL: 1},
			want:    want{201, "http://localhost:8080/1"},
		},
		{
			name:    "Negative test with empty request",
			request: "",
			store:   &mockStorage{},
			want:    want{400, ""},
		},
		{
			name:    "Negative test with addURLErr",
			request: "http://test.ru",
			store:   &mockStorage{addURLErr: errors.New("Negative test with addURLErr")},
			want:    want{500, ""},
		},
		{
			name:    "Test with 409 status",
			request: "http://test.ru",
			store:   &mockStorage{addURLErr: storage.ErrAlreadyExists, getURLID: 1},
			want:    want{409, "http://localhost:8080/1"},
		},
	}
	cfg := internal.Config{Address: ":8080", BaseURL: "http://localhost:8080"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store, cfg, Signer{SecretKey: []byte("sikrit")}, service.URLService{Store: tt.store})
			ts := httptest.NewServer(r)
			defer ts.Close()

			request := httptest.NewRequest(http.MethodPost, ts.URL+"/", bytes.NewBufferString(tt.request))
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, request)

			assert.Equal(t, tt.want.statusCode, resp.Code)
			if tt.want.url != "" {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.want.url, string(respBody))
			}
		})
	}

}

func TestShorten(t *testing.T) {
	type want struct {
		statusCode int
		resp       *ShortenResponse
	}
	tests := []struct {
		name    string
		request string
		store   storage.Storage
		want    want
	}{
		{
			name:    "Positive test",
			request: "{\"url\":\"http://test.ru\"}",
			store:   &mockStorage{addURL: 1},
			want:    want{201, &ShortenResponse{Result: "http://localhost:8080/1"}},
		},
		{
			name:    "Negative test with empty request",
			request: "",
			store:   &mockStorage{},
			want:    want{statusCode: 400},
		},
		{
			name:    "Negative test with addURLErr",
			request: "{\"url\":\"http://test.ru\"}",
			store:   &mockStorage{addURLErr: errors.New("Negative test with addURLErr")},
			want:    want{statusCode: 500},
		},
		{
			name:    "Test with 409 status",
			request: "{\"url\":\"http://test.ru\"}",
			store:   &mockStorage{addURLErr: storage.ErrAlreadyExists, getURLID: 1},
			want:    want{409, &ShortenResponse{Result: "http://localhost:8080/1"}},
		},
	}
	cfg := internal.Config{Address: ":8080", BaseURL: "http://localhost:8080"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store, cfg, Signer{SecretKey: []byte("secret")}, service.URLService{Store: tt.store})
			ts := httptest.NewServer(r)
			defer ts.Close()

			request := httptest.NewRequest(http.MethodPost, ts.URL+"/api/shorten", bytes.NewBufferString(tt.request))
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, request)

			assert.Equal(t, tt.want.statusCode, resp.Code)
			if tt.want.resp != nil {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				var result ShortenResponse
				err = json.Unmarshal(respBody, &result)
				require.NoError(t, err)
				assert.Equal(t, *tt.want.resp, result)
			}
		})
	}

}

func TestGetUserUrls(t *testing.T) {
	type want struct {
		statusCode int
		resp       []ShortOriginalURL
	}
	signer := Signer{SecretKey: []byte("another secret key")}
	token, err := signer.CreateSign(8)
	require.NoError(t, err)
	tests := []struct {
		name  string
		token string
		store storage.Storage
		want  want
	}{
		{
			name:  "Positive test",
			token: token,
			store: &mockStorage{userUrlsEmpty: false},
			want: want{200, []ShortOriginalURL{
				{ShortURL: "http://localhost:8080/8", OriginalURL: "http://test1.ru"},
				{ShortURL: "http://localhost:8080/9", OriginalURL: "http://test2.ru"},
			},
			},
		},
		{
			name:  "Negative test with empty token",
			token: "",
			store: &mockStorage{},
			want:  want{statusCode: 204},
		},
		{
			name:  "Negative test with empty urls",
			token: token,
			store: &mockStorage{userUrlsEmpty: true},
			want:  want{statusCode: 204},
		},
	}
	cfg := internal.Config{Address: ":8080", BaseURL: "http://localhost:8080"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store, cfg, signer, service.URLService{Store: tt.store})
			ts := httptest.NewServer(r)
			defer ts.Close()

			request := httptest.NewRequest(http.MethodGet, ts.URL+"/api/user/urls", nil)
			if tt.token != "" {
				cookie := &http.Cookie{Name: "token", Value: tt.token, MaxAge: 0}
				request.AddCookie(cookie)
			}
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, request)

			assert.Equal(t, tt.want.statusCode, resp.Code)
			if tt.want.resp != nil {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				var result []ShortOriginalURL
				err = json.Unmarshal(respBody, &result)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.want.resp, result)
			}
		})
	}

}

func TestShortenBatch(t *testing.T) {
	type want struct {
		statusCode int
		resp       []CorrIDShortURL
	}
	tests := []struct {
		name    string
		request string
		store   storage.Storage
		want    want
	}{
		{
			name:    "Positive test",
			request: "[{\"correlation_id\":\"str1\",\"original_url\":\"https://test1.ru\"},{\"correlation_id\":\"str2\",\"original_url\":\"https://test2.ru\"}]",
			store: &mockStorage{addBatch: []internal.CorrIDUrlID{
				{CorrID: "str1", URLID: 1},
				{CorrID: "str2", URLID: 2},
			}},
			want: want{201, []CorrIDShortURL{
				{CorrID: "str1", ShortURL: "http://localhost:8080/1"},
				{CorrID: "str2", ShortURL: "http://localhost:8080/2"},
			}},
		},
		{
			name:    "Negative test with empty request",
			request: "",
			store:   &mockStorage{},
			want:    want{statusCode: 400},
		},
		{
			name:    "Negative test with addBatchErr",
			request: "[{\"correlation_id\": \"str1\",\"original_url\": \"https://test1.ru\"}]",
			store:   &mockStorage{addBatchErr: errors.New("Negative test with addBatchErr")},
			want:    want{statusCode: 500},
		},
	}
	cfg := internal.Config{Address: ":8080", BaseURL: "http://localhost:8080"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store, cfg, Signer{SecretKey: []byte("my secret key")}, service.URLService{Store: tt.store})
			ts := httptest.NewServer(r)
			defer ts.Close()

			request := httptest.NewRequest(http.MethodPost, ts.URL+"/api/shorten/batch", bytes.NewBufferString(tt.request))
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, request)

			assert.Equal(t, tt.want.statusCode, resp.Code)
			if tt.want.resp != nil {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				var result []CorrIDShortURL
				err = json.Unmarshal(respBody, &result)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.want.resp, result)
			}
		})
	}

}

type mockStorage struct {
	addURL        int
	addURLErr     error
	getURL        string
	getURLErr     error
	getURLID      int
	getURLIDErr   error
	userUrlsEmpty bool
	addBatch      []internal.CorrIDUrlID
	addBatchErr   error
}

func (s *mockStorage) AddBatch(_ context.Context, _ []internal.CorrIDOriginalURL, _ int) ([]internal.CorrIDUrlID, error) {
	return s.addBatch, s.addBatchErr
}

func (s *mockStorage) GetUserUrls(_ context.Context, userID int) (map[int]string, error) {
	if s.userUrlsEmpty {
		return nil, storage.ErrNotFound
	}
	res := make(map[int]string)
	res[userID] = "http://test1.ru"
	res[userID+1] = "http://test2.ru"
	return res, nil
}

func (s *mockStorage) AddUser(_ context.Context) (int, error) {
	return 0, nil
}

func (s *mockStorage) AddURL(_ context.Context, _ string, _ int) (int, error) {
	return s.addURL, s.addURLErr
}

func (s *mockStorage) GetURLID(_ context.Context, _ string) (int, error) {
	return s.getURLID, s.getURLIDErr
}

func (s *mockStorage) GetURL(_ context.Context, _ string) (string, error) {
	return s.getURL, s.getURLErr
}

func (s *mockStorage) Close() {
}
