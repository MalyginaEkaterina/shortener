package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/service"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/caarlos0/env/v6"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

// Start parses flags and env vars and starts the server.
func Start() {
	var cfg internal.Config
	var pprofAddress string
	var secretFilePath string
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "address to listen on")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base address for short URL")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flag.StringVar(&secretFilePath, "p", "", "path to file with secret")
	flag.StringVar(&pprofAddress, "pprof", "localhost:6060", "address to export pprof on")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable https")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("Error while parsing env", err)
	}

	if pprofAddress != "" {
		go http.ListenAndServe(pprofAddress, nil)
	}

	store := initStore(cfg)
	defer store.Close()

	secretKey, err := getSecret(secretFilePath)
	if err != nil {
		log.Fatal("Error while reading secret key", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signer := handlers.Signer{SecretKey: secretKey}
	urlService := service.URLService{Store: store}
	deleteWorker := service.NewDeleteWorker(store)
	go deleteWorker.Run(ctx)
	r := handlers.NewRouter(store, cfg, signer, urlService, deleteWorker)

	if cfg.EnableHTTPS {
		log.Printf("Started TLS server on %s\n", cfg.Address)
		cert := generateTLSCertificate()
		server := &http.Server{
			Addr:    cfg.Address,
			Handler: r,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Printf("Started server on %s\n", cfg.Address)
		log.Fatal(http.ListenAndServe(cfg.Address, r))
	}
}

func generateTLSCertificate() tls.Certificate {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  privateKey,
		Leaf:        cert,
	}
}

func initStore(cfg internal.Config) storage.Storage {
	var store storage.Storage
	var err error
	if cfg.DatabaseDSN != "" {
		store, err = storage.NewDBStorage(cfg.DatabaseDSN)
		if err != nil {
			log.Fatal("Database connection error", err)
		}
		log.Printf("Using database storage %s\n", cfg.DatabaseDSN)
	} else if cfg.FileStoragePath != "" {
		store, err = storage.NewCachedFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Fatal("Error creating CachedFileStorage", err)
		}
		log.Printf("Using cached file storage %s\n", cfg.FileStoragePath)
	} else {
		store = storage.NewMemoryStorage()
		log.Printf("Using memory storage\n")
	}
	return store
}

func getSecret(path string) ([]byte, error) {
	if path == "" {
		// Only for tests.
		return []byte("my secret key"), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
