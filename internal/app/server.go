package app

import (
	"io"
	"log"
	"net/http"
	"strconv"
)

var urls = make([]string, 0)

func Start() {
	http.HandleFunc("/", short)
	server := &http.Server{
		Addr: ":8080",
	}
	log.Println("Server started")
	log.Fatal(server.ListenAndServe())
}

func short(writer http.ResponseWriter, req *http.Request) {
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
		urls = append(urls, string(body))
		resp := "http://localhost:8080/" + strconv.Itoa(len(urls)-1)
		writer.Header().Set("content-type", "text/html; charset=UTF-8")
		writer.WriteHeader(http.StatusCreated)
		_, err = writer.Write([]byte(resp))
		if err != nil {
			log.Println(err.Error())
		}
	} else if req.Method == http.MethodGet {
		strID := req.URL.Path[1:]
		if strID == "" {
			http.Error(writer, "Url ID is required", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(strID)
		if err != nil || id >= len(urls) {
			http.Error(writer, "Wrong URL ID", http.StatusBadRequest)
			return
		}
		url := urls[id]
		writer.Header().Set("Content-Type", "text/html; charset=UTF-8")
		writer.Header().Set("Location", url)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(writer, "Only GET and POST requests are allowed!", http.StatusBadRequest)
		return
	}
}
