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

func TestShortHandlerGet(t *testing.T) {
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
			name:    "Positive test with correct url id",
			request: "http://localhost:8080/1",
			urls:    storage.Storage{Urls: []string{"http://test1.ru", "http://test2.ru", "http://test3.ru"}},
			want:    want{307, "http://test2.ru"},
		},
		{
			name:    "Negative test with incorrect url id #1",
			request: "http://localhost:8080/1",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
		{
			name:    "Negative test with incorrect url id #2",
			request: "http://localhost:8080/abc",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
		{
			name:    "Negative test with empty url id",
			request: "http://localhost:8080/",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
		{
			name:    "Negative test with empty url id #2",
			request: "http://localhost:8080",
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			h := ShortHandler(tt.urls)
			h(w, request)
			result := w.Result()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			if tt.want.url != "" {
				assert.Equal(t, tt.want.url, result.Header.Get("Location"))
			}
			err := result.Body.Close()
			require.NoError(t, err)
		})
	}

}

func TestShortHandlerPost(t *testing.T) {
	type want struct {
		statusCode int
		url        string
	}
	tests := []struct {
		name    string
		request string
		method  string
		urls    storage.Storage
		want    want
	}{
		{
			name:    "Positive test",
			request: "http://test2.ru",
			method:  http.MethodPost,
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{201, "http://localhost:8080/1"},
		},
		{
			name:    "Positive test with empty urls",
			request: "http://test2.ru",
			method:  http.MethodPost,
			urls:    storage.Storage{},
			want:    want{201, "http://localhost:8080/0"},
		},
		{
			name:    "Negative test with empty request",
			request: "",
			method:  http.MethodPost,
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
		{
			name:    "Negative test with incorrect method",
			request: "http://test2.ru",
			method:  http.MethodPut,
			urls:    storage.Storage{Urls: []string{"http://test1.ru"}},
			want:    want{400, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, "http://localhost:8080/", bytes.NewBufferString(tt.request))
			w := httptest.NewRecorder()
			h := ShortHandler(tt.urls)
			h(w, request)
			result := w.Result()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			if tt.want.url != "" {
				resp, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.want.url, string(resp))
			}
			err := result.Body.Close()
			require.NoError(t, err)
		})
	}

}
