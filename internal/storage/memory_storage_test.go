package storage

import (
	"context"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"testing"
)

func BenchmarkGetUrl(b *testing.B) {
	store := NewMemoryStorage()
	userCount := 1000
	urlCount := 10000
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
	store := NewMemoryStorage()
	userCount := 1000
	urlCount := 10000
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
	tests := []struct {
		name      string
		sizeBatch int
		url       string
	}{
		{
			name:      "Batch 1000",
			sizeBatch: 1000,
			url:       "https://1000ya%d%d.ru",
		},
		{
			name:      "Batch 100",
			sizeBatch: 100,
			url:       "https://100ya%d%d.ru",
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			store := NewMemoryStorage()
			userCount := 1000
			urlCount := 10000
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
