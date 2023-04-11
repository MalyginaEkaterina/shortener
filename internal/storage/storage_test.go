package storage

import (
	"context"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

func BenchmarkGetUrl(b *testing.B) {
	userCount := 1000
	urlCount := 10000
	b.Run("memory storage", func(b *testing.B) {
		store := NewMemoryStorage()
		getURL(b, store, userCount, urlCount)
	})
	b.Run("file storage", func(b *testing.B) {
		f, err := os.CreateTemp("", "storage_test")
		require.NoError(b, err)
		defer os.Remove(f.Name())
		store, err := NewCachedFileStorage(f.Name())
		require.NoError(b, err)
		getURL(b, store, userCount, urlCount)
	})
}

func getURL(b *testing.B, store Storage, userCount, urlCount int) {
	urlIds := make([]int, urlCount)
	for i := 0; i < userCount; i++ {
		_, err := store.AddUser(context.Background())
		require.NoError(b, err)
	}
	for i := 0; i < urlCount; i++ {
		url := fmt.Sprintf("https://ya%d.ru", i)
		var err error
		urlIds[i], err = store.AddURL(context.Background(), url, rand.Intn(userCount))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := store.GetURL(context.Background(), strconv.Itoa(urlIds[rand.Intn(urlCount)]))
		require.NoError(b, err)
	}
}

func BenchmarkAddUrl(b *testing.B) {
	userCount := 1000
	urlCount := 10000
	b.Run("memory storage", func(b *testing.B) {
		store := NewMemoryStorage()
		addURL(b, store, userCount, urlCount)
	})
	b.Run("file storage", func(b *testing.B) {
		f, err := os.CreateTemp("", "storage_test")
		require.NoError(b, err)
		defer os.Remove(f.Name())
		store, err := NewCachedFileStorage(f.Name())
		require.NoError(b, err)
		addURL(b, store, userCount, urlCount)
	})
}

func addURL(b *testing.B, store Storage, userCount, urlCount int) {
	urlIds := make([]int, urlCount)
	for i := 0; i < userCount; i++ {
		_, err := store.AddUser(context.Background())
		require.NoError(b, err)
	}
	for i := 0; i < urlCount; i++ {
		url := fmt.Sprintf("https://ya%d.ru", i)
		var err error
		urlIds[i], err = store.AddURL(context.Background(), url, rand.Intn(userCount))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		url := fmt.Sprintf("https://da%d.ru", rand.Intn(10000))
		_, err := store.AddURL(context.Background(), url, rand.Intn(userCount))
		if err != nil {
			assert.Equal(b, ErrAlreadyExists, err)
		}
	}
}

func BenchmarkAddBatch(b *testing.B) {
	var toRemove []string
	defer func() {
		for _, name := range toRemove {
			os.Remove(name)
		}
	}()
	tests := []struct {
		name      string
		sizeBatch int
		url       string
		storeNew  func() (Storage, error)
	}{
		{
			name:      "Memory storage. Batch 1000",
			sizeBatch: 1000,
			url:       "https://1000ya%d%d.ru",
			storeNew: func() (Storage, error) {
				return NewMemoryStorage(), nil
			},
		},
		{
			name:      "Memory storage. Batch 100",
			sizeBatch: 100,
			url:       "https://100ya%d%d.ru",
			storeNew: func() (Storage, error) {
				return NewMemoryStorage(), nil
			},
		},
		{
			name:      "File storage. Batch 1000",
			sizeBatch: 1000,
			url:       "https://1000ya%d%d.ru",
			storeNew: func() (Storage, error) {
				f, err := os.CreateTemp("", "storage_test")
				require.NoError(b, err)
				toRemove = append(toRemove, f.Name())
				return NewCachedFileStorage(f.Name())
			},
		},
		{
			name:      "File storage. Batch 100",
			sizeBatch: 100,
			url:       "https://100ya%d%d.ru",
			storeNew: func() (Storage, error) {
				f, err := os.CreateTemp("", "storage_test")
				require.NoError(b, err)
				toRemove = append(toRemove, f.Name())
				return NewCachedFileStorage(f.Name())
			},
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			userCount := 1000
			urlCount := 10000
			store, err := tt.storeNew()
			require.NoError(b, err)
			urlIds := make([]int, urlCount)
			for i := 0; i < userCount; i++ {
				_, err := store.AddUser(context.Background())
				require.NoError(b, err)
			}
			for i := 0; i < urlCount; i++ {
				url := fmt.Sprintf("https://ya%d.ru", i)
				var err error
				urlIds[i], err = store.AddURL(context.Background(), url, rand.Intn(userCount))
				require.NoError(b, err)
			}
			urls := make([]internal.CorrIDOriginalURL, tt.sizeBatch)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				for j := 0; j < tt.sizeBatch; j++ {
					url := fmt.Sprintf(tt.url, j, rand.Intn(1000))
					urls[j] = internal.CorrIDOriginalURL{CorrID: strconv.Itoa(j), OriginalURL: url}
				}
				b.StartTimer()
				_, err := store.AddBatch(context.Background(), urls, rand.Intn(userCount))
				require.NoError(b, err)
			}
		})
	}
}
