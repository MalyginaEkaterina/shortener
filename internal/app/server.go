package app

import (
	"flag"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/caarlos0/env/v6"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
)

func Start() {
	var cfg internal.Config
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "address to listen on")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base address for short URL")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal("Error while parsing env", err)
	}
	var store storage.Storage
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
		store = &storage.MemoryStorage{UserUrls: make(map[int][]int), UrlsID: make(map[string]int)}
		log.Printf("Using memory storage\n")
	}
	defer store.Close()
	r := handlers.NewRouter(store, cfg)
	log.Printf("Started server on %s\n", cfg.Address)
	log.Fatal(http.ListenAndServe(cfg.Address, r))
}
