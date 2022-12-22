package app

import (
	"github.com/MalyginaEkaterina/shortener/internal/handlers"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"log"
	"net/http"
)

func Start() {
	urls := storage.Storage{}
	http.HandleFunc("/", handlers.ShortHandler(urls))
	server := &http.Server{
		Addr: ":8080",
	}
	log.Println("Server started")
	log.Fatal(server.ListenAndServe())
}
