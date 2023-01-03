package app

import (
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
	"net/http"
)

func Start() {
	store := storage.MemoryStorage{}
	r := handlers.NewRouter(&store)
	log.Fatal(http.ListenAndServe(":8080", r))
}
