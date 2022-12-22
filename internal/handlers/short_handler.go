package handlers

import (
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"io"
	"log"
	"net/http"
	"strconv"
)

func ShortHandler(urls storage.Storage) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
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
		} else if req.Method == http.MethodGet {
			if len(req.URL.Path) <= 1 {
				http.Error(writer, "Url ID is required", http.StatusBadRequest)
				return
			}
			strID := req.URL.Path[1:]
			id, err := strconv.Atoi(strID)
			if err != nil || !urls.ValidID(id) {
				http.Error(writer, "Wrong URL ID", http.StatusBadRequest)
				return
			}
			url := urls.GetURL(id)
			writer.Header().Set("Content-Type", "text/html; charset=UTF-8")
			writer.Header().Set("Location", url)
			writer.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.Error(writer, "Only GET and POST requests are allowed!", http.StatusBadRequest)
			return
		}
	}
}
