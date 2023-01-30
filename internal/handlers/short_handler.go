package handlers

import (
	"encoding/json"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"strconv"
)

func NewRouter(store storage.Storage, cfg internal.Config) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(gzipHandle)

	r.Route("/", func(r chi.Router) {
		r.Post("/", ShortURL(store, cfg.BaseURL))
		r.Get("/{id}", GetURLByID(store))
		r.Post("/api/shorten", Shorten(store, cfg.BaseURL))
		r.Get("/api/user/urls", GetUserUrls(store, cfg.BaseURL))
	})

	r.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Wrong request", http.StatusBadRequest)
	})

	r.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Method not allowed", http.StatusBadRequest)
	})
	return r
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type ShortOriginalURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var (
	ErrSignNotValid = errors.New("sign is not valid")
)

func getIDAndCookie(store storage.Storage, req *http.Request) (int, *http.Cookie, error) {
	var userID int
	var authOK bool
	var signValue string
	var cookie *http.Cookie

	sign, err := req.Cookie("token")
	if err == nil {
		signValue = sign.Value
		userID, authOK, err = CheckSign(signValue)
		if err != nil {
			log.Println("Error while checking of sign", err)
			return 0, nil, err
		}
	}
	if err != nil || !authOK {
		userID, err = store.AddUser()
		if err != nil {
			log.Println("Error while adding user", err)
			return 0, nil, err
		}
		signValue, err = CreateSign(userID)
		if err != nil {
			log.Println("Error while creating of sign", err)
			return 0, nil, err
		}
		cookie = &http.Cookie{Name: "token", Value: signValue, MaxAge: 0}
	}
	return userID, cookie, nil
}

func getID(req *http.Request) (int, error) {
	sign, err := req.Cookie("token")
	if err != nil {
		return 0, err
	}
	userID, authOK, err := CheckSign(sign.Value)
	if err != nil {
		log.Println("Error while checking of sign", err)
		return 0, err
	}
	if !authOK {
		return 0, ErrSignNotValid
	}
	return userID, nil
}

func Shorten(store storage.Storage, baseURL string) http.HandlerFunc {
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
		var shortenRequest ShortenRequest
		err = json.Unmarshal(body, &shortenRequest)
		if err != nil {
			http.Error(writer, "Failed to parse request body", http.StatusBadRequest)
			return
		}
		userID, tokenCookie, err := getIDAndCookie(store, req)
		if err != nil {
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		ind, err := store.AddURL(shortenRequest.URL, userID)
		if err != nil {
			log.Println("Error while adding URl", err)
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		response := ShortenResponse{Result: baseURL + "/" + strconv.Itoa(ind)}
		respJSON, err := json.Marshal(response)
		if err != nil {
			log.Println("Error while serializing response", err)
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		if tokenCookie != nil {
			http.SetCookie(writer, tokenCookie)
		}
		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		writer.Write(respJSON)
	}
}

func ShortURL(store storage.Storage, baseURL string) http.HandlerFunc {
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
		userID, tokenCookie, err := getIDAndCookie(store, req)
		if err != nil {
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		ind, err := store.AddURL(string(body), userID)
		if err != nil {
			log.Println("Error while adding URl", err)
			http.Error(writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		resp := baseURL + "/" + strconv.Itoa(ind)
		if tokenCookie != nil {
			http.SetCookie(writer, tokenCookie)
		}
		writer.Header().Set("content-type", "text/html; charset=UTF-8")
		writer.WriteHeader(http.StatusCreated)
		writer.Write([]byte(resp))
	}
}

func GetURLByID(store storage.Storage) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if id == "" {
			http.Error(writer, "Url ID is required", http.StatusBadRequest)
			return
		}
		url, err := store.GetURL(id)
		if err != nil {
			if err == storage.ErrNotFound {
				http.Error(writer, "Not found", http.StatusBadRequest)
			} else {
				log.Println("Error while getting URL", err)
				http.Error(writer, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=UTF-8")
		writer.Header().Set("Location", url)
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func GetUserUrls(store storage.Storage, baseURL string) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		userID, err := getID(req)
		if err != nil {
			//http.Error(writer, "Failed to check sign", http.StatusBadRequest)
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		urls, err := store.GetUserUrls(userID)
		if err != nil {
			if err == storage.ErrNotFound {
				writer.WriteHeader(http.StatusNoContent)
			} else {
				log.Println("Error while getting URLs", err)
				http.Error(writer, "Internal server error", http.StatusInternalServerError)
			}
		} else {
			var urlsList []ShortOriginalURL
			for urlID, originalURL := range urls {
				urlsList = append(urlsList, ShortOriginalURL{ShortURL: baseURL + "/" + strconv.Itoa(urlID), OriginalURL: originalURL})
			}
			respJSON, err := json.Marshal(urlsList)
			if err != nil {
				log.Println("Error while serializing response", err)
				http.Error(writer, "Internal server error", http.StatusInternalServerError)
				return
			}
			writer.Header().Set("content-type", "application/json")
			writer.WriteHeader(http.StatusOK)
			writer.Write(respJSON)
		}
	}
}
