package handlers

import (
	"encoding/json"
	"errors"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/service"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"strconv"
)

type Router struct {
	store   storage.Storage
	signer  Signer
	baseURL string
	service service.Service
}

func NewRouter(store storage.Storage, cfg internal.Config, signer Signer, service service.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(gzipHandle)

	router := &Router{
		store:   store,
		signer:  signer,
		baseURL: cfg.BaseURL,
		service: service,
	}

	r.Route("/", func(r chi.Router) {
		r.Post("/", router.ShortURL)
		r.Get("/{id}", router.GetURLByID)
		r.Post("/api/shorten", router.Shorten)
		r.Get("/api/user/urls", router.GetUserUrls)
		r.Get("/ping", PingDB(store))
		r.Post("/api/shorten/batch", router.ShortenBatch)
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

type CorrIDShortURL struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

var (
	ErrSignNotValid = errors.New("sign is not valid")
)

func (r *Router) getIDAndCookie(req *http.Request) (int, *http.Cookie, error) {
	var userID int
	var authOK bool
	var signValue string
	var cookie *http.Cookie

	sign, err := req.Cookie("token")
	if err == nil {
		signValue = sign.Value
		userID, authOK, err = r.signer.CheckSign(signValue)
		if err != nil {
			log.Println("Error while checking of sign", err)
			return 0, nil, err
		}
	}
	if err != nil || !authOK {
		userID, err = r.store.AddUser(req.Context())
		if err != nil {
			log.Println("Error while adding user", err)
			return 0, nil, err
		}
		signValue, err = r.signer.CreateSign(userID)
		if err != nil {
			log.Println("Error while creating of sign", err)
			return 0, nil, err
		}
		cookie = &http.Cookie{Name: "token", Value: signValue, MaxAge: 0}
	}
	return userID, cookie, nil
}

func (r *Router) getID(req *http.Request) (int, error) {
	sign, err := req.Cookie("token")
	if err != nil {
		return 0, err
	}
	userID, authOK, err := r.signer.CheckSign(sign.Value)
	if err != nil {
		log.Println("Error while checking of sign", err)
		return 0, err
	}
	if !authOK {
		return 0, ErrSignNotValid
	}
	return userID, nil
}

func (r *Router) Shorten(writer http.ResponseWriter, req *http.Request) {
	var shortenRequest ShortenRequest
	if !unmarshalRequestJSON(writer, req, &shortenRequest) {
		return
	}
	userID, tokenCookie, err := r.getIDAndCookie(req)
	if err != nil {
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	ind, alreadyExists, err := r.service.AddURL(req.Context(), shortenRequest.URL, userID)
	if err != nil {
		log.Println("Error while adding URl", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	var status int
	if alreadyExists {
		status = http.StatusConflict
	} else {
		status = http.StatusCreated
	}
	response := ShortenResponse{Result: r.baseURL + "/" + strconv.Itoa(ind)}
	marshalResponseJSON(writer, status, tokenCookie, response)
}

func marshalResponseJSON(writer http.ResponseWriter, status int, cookie *http.Cookie, response any) {
	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Println("Error while serializing response", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	if cookie != nil {
		http.SetCookie(writer, cookie)
	}
	writer.Header().Set("content-type", "application/json")
	writer.WriteHeader(status)
	writer.Write(respJSON)
}

func unmarshalRequestJSON(writer http.ResponseWriter, req *http.Request, v any) bool {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return false
	}
	if len(body) == 0 {
		http.Error(writer, "Request body is required", http.StatusBadRequest)
		return false
	}
	err = json.Unmarshal(body, v)
	if err != nil {
		http.Error(writer, "Failed to parse request body", http.StatusBadRequest)
		return false
	}
	return true
}

func (r *Router) ShortURL(writer http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(writer, "Request body is required", http.StatusBadRequest)
		return
	}
	userID, tokenCookie, err := r.getIDAndCookie(req)
	if err != nil {
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	url := string(body)
	ind, alreadyExists, err := r.service.AddURL(req.Context(), url, userID)
	if err != nil {
		log.Println("Error while adding URl", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	var status int
	if alreadyExists {
		status = http.StatusConflict
	} else {
		status = http.StatusCreated
	}
	resp := r.baseURL + "/" + strconv.Itoa(ind)
	if tokenCookie != nil {
		http.SetCookie(writer, tokenCookie)
	}
	writer.Header().Set("content-type", "text/html; charset=UTF-8")
	writer.WriteHeader(status)
	writer.Write([]byte(resp))
}

func (r *Router) GetURLByID(writer http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		http.Error(writer, "Url ID is required", http.StatusBadRequest)
		return
	}
	url, err := r.store.GetURL(req.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
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

func (r *Router) GetUserUrls(writer http.ResponseWriter, req *http.Request) {
	userID, err := r.getID(req)
	if err != nil {
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	urls, err := r.store.GetUserUrls(req.Context(), userID)
	if errors.Is(err, storage.ErrNotFound) || len(urls) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		return
	} else if err != nil {
		log.Println("Error while getting URLs", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	var urlsList []ShortOriginalURL
	for urlID, originalURL := range urls {
		urlsList = append(urlsList, ShortOriginalURL{ShortURL: r.baseURL + "/" + strconv.Itoa(urlID), OriginalURL: originalURL})
	}
	marshalResponseJSON(writer, http.StatusOK, nil, urlsList)
}

func (r *Router) ShortenBatch(writer http.ResponseWriter, req *http.Request) {
	var urls []internal.CorrIDOriginalURL
	if !unmarshalRequestJSON(writer, req, &urls) {
		return
	}
	userID, tokenCookie, err := r.getIDAndCookie(req)
	if err != nil {
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	corrIDUrlIDs, err := r.store.AddBatch(req.Context(), urls, userID)
	if err != nil {
		log.Println("Error while adding URls", err)
		http.Error(writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	shortenUrls := make([]CorrIDShortURL, len(corrIDUrlIDs))
	for i, v := range corrIDUrlIDs {
		u := CorrIDShortURL{CorrID: v.CorrID, ShortURL: r.baseURL + "/" + strconv.Itoa(v.URLID)}
		shortenUrls[i] = u
	}
	marshalResponseJSON(writer, http.StatusCreated, tokenCookie, shortenUrls)
}
