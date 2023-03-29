package handlers

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func BenchmarkCheckSign(b *testing.B) {
	signer := Signer{SecretKey: []byte("secret key")}
	signs := make([]string, 1000)
	for i := range signs {
		var err error
		signs[i], err = signer.CreateSign(i)
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := signer.CheckSign(signs[rand.Intn(len(signs))])
		require.NoError(b, err)
	}
}

func BenchmarkCreateSign(b *testing.B) {
	signer := Signer{SecretKey: []byte("secret key")}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := signer.CreateSign(rand.Intn(100000))
		require.NoError(b, err)
	}
}
