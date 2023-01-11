package app

import (
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/caarlos0/env/v6"
	"log"
	"net/http"
)

func Start() {
	store := storage.MemoryStorage{}
	var cfg internal.Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	r := handlers.NewRouter(&store, cfg)
	log.Fatal(http.ListenAndServe(cfg.Address, r))
}
