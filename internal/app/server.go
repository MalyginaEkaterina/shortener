package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"flag"
	"fmt"
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
	"os/signal"
	"syscall"
	"time"
)

// Start parses flags and env vars and starts the server.
func Start() {
	cfg := internal.Config{
		Address: "localhost:8080",
		BaseURL: "http://localhost:8080",
	}

	appName := os.Args[0]
	cfgFlag := flag.NewFlagSet(appName, flag.ContinueOnError)
	var configName string
	cfgFlag.StringVar(&configName, "c", os.Getenv("CONFIG"), "name of config file")
	cfgFlag.Parse(os.Args[1:])
	if configName != "" {
		confData, err := os.ReadFile(configName)
		if err != nil {
			log.Fatal("Error while reading config file")
		}
		err = json.Unmarshal(confData, &cfg)
		if err != nil {
			log.Fatal("Error while parsing config file")
		}
	}

	var pprofAddress string
	var secretFilePath string
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.StringVar(&cfg.Address, "a", cfg.Address, "address to listen on")
	flags.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base address for short URL")
	flags.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flags.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database connection string")
	flags.BoolVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "enable https")
	flags.StringVar(&secretFilePath, "p", "", "path to file with secret")
	flags.StringVar(&pprofAddress, "pprof", "localhost:6060", "address to export pprof on")
	flags.StringVar(&configName, "c", os.Getenv("CONFIG"), "name of config file")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Fatal("Error parsing args", err)
	}

	if err := env.Parse(&cfg); err != nil {
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

	sigint := make(chan os.Signal, 1)
	connsClosed := make(chan struct{})
	signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	shutdown := func(srv *http.Server) {
		<-sigint
		if er := srv.Shutdown(context.Background()); er != nil {
			log.Printf("HTTP server Shutdown: %v", er)
		}
		close(connsClosed)
	}

	if cfg.EnableHTTPS {
		cert, err := generateTLSCertificate()
		if err != nil {
			log.Fatal(err)
		}
		server := &http.Server{
			Addr:    cfg.Address,
			Handler: r,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{*cert},
			},
		}
		go shutdown(server)
		log.Printf("Started TLS server on %s\n", cfg.Address)
		if err = server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServeTLS: %v", err)
		}
	} else {
		server := &http.Server{Addr: cfg.Address, Handler: r}
		go shutdown(server)
		log.Printf("Started server on %s\n", cfg.Address)
		if err = server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}
	<-connsClosed
	log.Printf("Stopped server on %s\n", cfg.Address)
}

func generateTLSCertificate() (*tls.Certificate, error) {
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
		return nil, fmt.Errorf(`error while key generating: %w`, err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf(`error while creating certificate: %w`, err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  privateKey,
		Leaf:        cert,
	}, nil
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
