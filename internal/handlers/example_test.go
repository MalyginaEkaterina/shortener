package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MalyginaEkaterina/shortener/internal"
	"github.com/MalyginaEkaterina/shortener/internal/service"
	"github.com/MalyginaEkaterina/shortener/internal/storage"
	"io"
	"log"
	"net/http"
	"strings"
)

type ShortenReq struct {
	URL string `json:"url"`
}

type ShortenResp struct {
	Result string `json:"result"`
}

type GetUserURLResp struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ShortenBatchReq struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}

type ShortenBatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

func Example_shorten() {
	ts, serverURL := newTestServer()
	defer ts.Close()

	shortenedURL, _ := ShortenURL(serverURL, "http://ya.ru")
	fmt.Println("Shortened URL:", shortenedURL)
}

func Example_getURLByID() {
	ts, serverURL := newTestServer()
	defer ts.Close()

	shortenedURL, _ := ShortenURL(serverURL, "http://google.com")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest(http.MethodGet, shortenedURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	originalURL, err := response.Location()
	fmt.Println("Original URL:", originalURL.String())
}

func Example_getUserUrls() {
	ts, serverURL := newTestServer()
	defer ts.Close()

	_, token := ShortenURL(serverURL, "http://ya.ru")

	var client http.Client
	request, err := http.NewRequest(http.MethodGet, serverURL+"/api/user/urls", nil)
	if err != nil {
		log.Fatal(err)
	}
	cookie := &http.Cookie{Name: "token", Value: token, MaxAge: 0}
	request.AddCookie(cookie)
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	var getUserURLResp []GetUserURLResp
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &getUserURLResp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User's URLs:", getUserURLResp)
}

func Example_deleteBatch() {
	ts, serverURL := newTestServer()
	defer ts.Close()

	shortenedURL, token := ShortenURL(serverURL, "http://ya.ru")
	delURLS := []string{getURLID(shortenedURL)}
	delURLSjson, err := json.Marshal(delURLS)
	if err != nil {
		log.Fatal(err)
	}
	var client http.Client
	request, err := http.NewRequest(http.MethodDelete, serverURL+"/api/user/urls", bytes.NewBuffer(delURLSjson))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	cookie := &http.Cookie{Name: "token", Value: token, MaxAge: 0}
	request.AddCookie(cookie)
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	fmt.Println("Deletion status:", response.Status)
}

func Example_shortenBatch() {
	ts, serverURL := newTestServer()
	defer ts.Close()

	var client http.Client

	shortenBatchReq := []ShortenBatchReq{
		{CorrID: "1", OriginalURL: "https://ya1.ru"},
		{CorrID: "2", OriginalURL: "https://ya2.ru"},
	}

	shortenBatchReqJSON, err := json.Marshal(shortenBatchReq)
	if err != nil {
		log.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, serverURL+"/api/shorten/batch", bytes.NewBuffer(shortenBatchReqJSON))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	var shortenBatchResp []ShortenBatchResp
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &shortenBatchResp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Shortened URLs:", shortenBatchResp)
}

func GetTokenFromResponse(response *http.Response) string {
	for _, c := range response.Cookies() {
		if c.Name == "token" {
			return c.Value
		}
	}
	return ""
}

func ShortenURL(serviceURL string, url string) (shortenedURL string, token string) {
	var client http.Client

	shortenReq := ShortenReq{URL: url}
	shortenReqJSON, err := json.Marshal(shortenReq)
	if err != nil {
		log.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, serviceURL+"/api/shorten", bytes.NewBuffer(shortenReqJSON))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	shortenResp := ShortenResp{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &shortenResp)
	if err != nil {
		log.Fatal(err)
	}

	return shortenResp.Result, GetTokenFromResponse(response)
}

func newTestServer() (ts *http.Server, serverURL string) {
	store := storage.NewMemoryStorage()
	cfg := internal.Config{
		Address: ":8392",
		BaseURL: "http://localhost:8392",
	}
	r := NewRouter(store, cfg,
		service.URLService{Store: store, Signer: service.Signer{SecretKey: []byte("secret again")}},
		service.NewDeleteWorker(store))
	ts = &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}
	go ts.ListenAndServe()
	return ts, cfg.BaseURL
}

func getURLID(result string) string {
	idx := strings.LastIndexByte(result, '/')
	return result[idx+1:]
}
