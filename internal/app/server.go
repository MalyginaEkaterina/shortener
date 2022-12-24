package app

import (
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
	"net/http"
)

func Start() {
	urls := storage.Storage{}
	r := handlers.NewRouter(&urls)
	log.Fatal(http.ListenAndServe(":8080", r))
}
