package handlers

import (
	"bytes"
	"errors"
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
			store: &mockStorage{validID: true, getURL: "http://test.ru"},
			want:  want{307, "http://test.ru"},
		},
		{
			name:  "Negative test with incorrect url id #1",
			path:  "/1",
			store: &mockStorage{validID: false},
			want:  want{400, ""},
		},
		{
			name:  "Negative test with validError",
			path:  "/1",
			store: &mockStorage{validIDErr: errors.New("negative test with validError")},
			want:  want{500, ""},
		},
		{
			name:  "Negative test with getURLError",
			path:  "/1",
			store: &mockStorage{validID: true, getURLErr: errors.New("negative test with getURLError")},
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(tt.store)
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

type mockStorage struct {
	addURL     int
	addURLErr  error
	validID    bool
	validIDErr error
	getURL     string
	getURLErr  error
}

func (s *mockStorage) AddURL(_ string) (int, error) {
	return s.addURL, s.addURLErr
}

func (s *mockStorage) ValidID(_ string) (bool, error) {
	return s.validID, s.validIDErr
}

func (s *mockStorage) GetURL(_ string) (string, error) {
	return s.getURL, s.getURLErr
}
