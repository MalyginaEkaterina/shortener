package handlers

import (
	"bytes"
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
		name string
		path string
		urls storage.Storage
		want want
	}{
		{
			name: "Positive test with correct url id",
			path: "/1",
			urls: storage.Storage{Urls: []string{"http://test1.ru", "http://test2.ru", "http://test3.ru"}},
			want: want{307, "http://test2.ru"},
		},
		{
			name: "Negative test with incorrect url id #1",
			path: "/1",
			urls: storage.Storage{Urls: []string{"http://test1.ru"}},
			want: want{400, ""},
		},
		{
			name: "Negative test with incorrect url id #2",
			path: "/abc",
			urls: storage.Storage{Urls: []string{"http://test1.ru"}},
			want: want{400, ""},
		},
		{
			name: "Negative test with empty url id",
			path: "/",
			urls: storage.Storage{Urls: []string{"http://test1.ru"}},
			want: want{400, ""},
		},
		{
			name: "Negative test with empty url id #2",
			path: "",
			urls: storage.Storage{Urls: []string{"http://test1.ru"}},
			want: want{400, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(&tt.urls)
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
		urls    storage.Storage
		want    want
	}{
		{
			name:    "Positive test",
			request: "http://test2.ru",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{201, "http://localhost:8080/1"},
		},
		{
			name:    "Positive test with empty urls",
			request: "http://test2.ru",
			urls:    storage.Storage{},
			want:    want{201, "http://localhost:8080/0"},
		},
		{
			name:    "Negative test with empty request",
			request: "",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter(&tt.urls)
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
