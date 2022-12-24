package handlers

import (
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"strconv"
)

func NewRouter(urls *storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/", func(r chi.Router) {
		r.Post("/", ShortURL(urls))
		r.Get("/{id}", GetURLByID(urls))
	})

	r.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Wrong request", http.StatusBadRequest)
	})

	r.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Method not allowed", http.StatusBadRequest)
	})
	return r
}

func ShortURL(urls *storage.Storage) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			http.Error(writer, "Request body is required", http.StatusBadRequest)
			return
		}
		ind := urls.AddURL(string(body))
		resp := "http://localhost:8080/" + strconv.Itoa(ind)
		writer.Header().Set("content-type", "text/html; charset=UTF-8")
		writer.WriteHeader(http.StatusCreated)
		_, err = writer.Write([]byte(resp))
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func GetURLByID(urls *storage.Storage) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		idStr := chi.URLParam(req, "id")
		if idStr == "" {
			http.Error(writer, "Url ID is required", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil || !urls.ValidID(id) {
			http.Error(writer, "Wrong URL ID", http.StatusBadRequest)
			return
		}
		url := urls.GetURL(id)
		writer.Header().Set("Content-Type", "text/html; charset=UTF-8")
		writer.Header().Set("Location", url)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}
}
