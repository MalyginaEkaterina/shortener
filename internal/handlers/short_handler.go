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
	"net"
	"net/http"
	"strconv"
)

// Router routes http requests.
type Router struct {
	store         storage.Storage
	baseURL       string
	service       service.Service
	deleteWorker  service.DeleteWorker
	trustedSubnet *net.IPNet
}

// NewRouter creates new chi Router and configures it.
func NewRouter(store storage.Storage, cfg internal.Config, service service.Service, deleteWorker service.DeleteWorker) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(gzipHandle)

	router := &Router{
		store:         store,
		baseURL:       cfg.BaseURL,
		service:       service,
		deleteWorker:  deleteWorker,
		trustedSubnet: cfg.TrustedSubnet,
	}

	r.Route("/", func(r chi.Router) {
		r.Post("/", router.ShortURL)
		r.Get("/{id}", router.GetURLByID)
		r.Post("/api/shorten", router.Shorten)
		r.Get("/api/user/urls", router.GetUserUrls)
		r.Get("/ping", PingDB(store))
		r.Post("/api/shorten/batch", router.ShortenBatch)
		r.Delete("/api/user/urls", router.DeleteBatch)
		r.Get("/api/internal/stats", router.GetStats)
	})

	r.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Wrong request", http.StatusBadRequest)
	})

	r.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Method not allowed", http.StatusBadRequest)
	})
	return r
}

// ShortenRequest contains a request to shorten the URL.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse contains a response with shortened URL.
type ShortenResponse struct {
	Result string `json:"result"`
}

// ShortOriginalURL contains original URL and its corresponding shortened URL.
type ShortOriginalURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// CorrIDShortURL contains shortened URL and its corresponding correlation_id from the request to shorten of the batch.
type CorrIDShortURL struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

// Stat contains count of URLs and count of Users
type Stat struct {
	URLCount   int `json:"urls"`
	UsersCount int `json:"users"`
}

// Consts
const (
	IPHeader = "X-Real-IP"
)

func (r *Router) getIDAndCookie(req *http.Request) (int, *http.Cookie, error) {
	var userID int
	var sign string
	var cookie *http.Cookie

	token, err := req.Cookie("token")
	if err == nil {
		sign = token.Value
	}
	userID, sign, err = r.service.GetUserIDOrCreate(req.Context(), sign)
	if err != nil {
		return 0, nil, err
	}
	cookie = &http.Cookie{Name: "token", Value: sign, MaxAge: 0}
	return userID, cookie, nil
}

// Shorten receives JSON with URL and returns status 201 and shortened URL.
// Returns status 409 and shortened URL if the URL has already been shortened.
// If request does not contain a valid token new user will be created.
func (r *Router) Shorten(writer http.ResponseWriter, req *http.Request) {
	var shortenRequest ShortenRequest
	if !unmarshalRequest(writer, req, &shortenRequest) {
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
	marshalResponseAndSetCookie(writer, status, tokenCookie, response)
}

func marshalResponseAndSetCookie(writer http.ResponseWriter, status int, cookie *http.Cookie, response any) {
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

func unmarshalRequest(writer http.ResponseWriter, req *http.Request, v any) bool {
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

// ShortURL receives text with URL and returns status 201 and shortened URL.
// Returns status 409 if the URL has already been shortened.
// If request does not contain a valid token new user will be created.
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

// GetURLByID receives url parameter with id and returns status 307 and associated URL in header Location.
// Returns status 400 if requested id does not exist.
// Returns status 410 if requested id was deleted.
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
		} else if errors.Is(err, storage.ErrDeleted) {
			http.Error(writer, "Was deleted", http.StatusGone)
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

// GetUserUrls returns the list of shortened and original URLs for the user.
// Returns status 204 if there is no data for the user.
func (r *Router) GetUserUrls(writer http.ResponseWriter, req *http.Request) {
	sign, err := req.Cookie("token")
	if err != nil {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	userID, err := r.service.GetUserID(sign.Value)
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
	marshalResponseAndSetCookie(writer, http.StatusOK, nil, urlsList)
}

// ShortenBatch receives JSON with the list of URLs and their correlation_id and returns status 201
// and the list of shortened URLs with their correlation_id.
// If request does not contain a valid token a new user will be created.
func (r *Router) ShortenBatch(writer http.ResponseWriter, req *http.Request) {
	var urls []internal.CorrIDOriginalURL
	if !unmarshalRequest(writer, req, &urls) {
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
	marshalResponseAndSetCookie(writer, http.StatusCreated, tokenCookie, shortenUrls)
}

// DeleteBatch receives the list of shortened URL IDs, queued them for deletion and returns status 202.
func (r *Router) DeleteBatch(writer http.ResponseWriter, req *http.Request) {
	var urlIDs []string
	if !unmarshalRequest(writer, req, &urlIDs) {
		return
	}

	sign, err := req.Cookie("token")
	if err != nil {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	userID, err := r.service.GetUserID(sign.Value)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	idsToDelete := make([]internal.IDToDelete, len(urlIDs))

	for i, idStr := range urlIDs {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		idsToDelete[i] = internal.IDToDelete{ID: id, UserID: userID}
	}
	r.deleteWorker.Delete(idsToDelete)
	writer.WriteHeader(http.StatusAccepted)
}

// GetStats returns statistics
func (r *Router) GetStats(writer http.ResponseWriter, req *http.Request) {
	if r.trustedSubnet == nil {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}
	ip := net.ParseIP(req.Header.Get(IPHeader))
	if !r.trustedSubnet.Contains(ip) {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}
	urls, users, err := r.store.GetStat(req.Context())
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	stat := Stat{
		URLCount:   urls,
		UsersCount: users,
	}
	marshalResponseAndSetCookie(writer, http.StatusOK, nil, stat)
}
