package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func Example() {
	const address = "http://localhost:8080"
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Shorten URL

	shortenReq := ShortenReq{URL: "https://ya.ru"}
	shortenReqJSON, err := json.Marshal(shortenReq)
	if err != nil {
		log.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, address+"/api/shorten", bytes.NewBuffer(shortenReqJSON))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	var token *http.Cookie
	for _, c := range response.Cookies() {
		if c.Name == "token" {
			token = c
			break
		}
	}
	fmt.Println(token)
	shortenResp := ShortenResp{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	err = json.Unmarshal(body, &shortenResp)
	fmt.Println("Shortened URL:", shortenResp.Result)
	shortenedURLID := getURLID(shortenResp.Result)

	// Get Original URL by id

	request, err = http.NewRequest(http.MethodGet, shortenResp.Result, nil)
	if err != nil {
		log.Fatal(err)
	}
	response, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	originalURL, err := response.Location()
	fmt.Println("Original URL:", originalURL.String())

	// Get user URLs

	request, err = http.NewRequest(http.MethodGet, address+"/api/user/urls", nil)
	if err != nil {
		log.Fatal(err)
	}
	request.AddCookie(token)
	response, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	var getUserURLResp []GetUserURLResp
	body, err = io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	err = json.Unmarshal(body, &getUserURLResp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User's URLs:", getUserURLResp)

	// Delete the batch of URLs

	delURLS := []string{shortenedURLID}
	delURLSjson, err := json.Marshal(delURLS)
	if err != nil {
		log.Fatal(err)
	}
	request, err = http.NewRequest(http.MethodDelete, address+"/api/user/urls", bytes.NewBuffer(delURLSjson))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	request.AddCookie(token)
	response, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	fmt.Println("Deletion status:", response.Status)

	// Shorten the batch of URLs

	shortenBatchReq := []ShortenBatchReq{
		{CorrID: "1", OriginalURL: "https://ya1.ru"},
		{CorrID: "2", OriginalURL: "https://ya2.ru"},
	}

	shortenBatchReqJSON, err := json.Marshal(shortenBatchReq)
	if err != nil {
		log.Fatal(err)
	}
	request, err = http.NewRequest(http.MethodPost, address+"/api/shorten/batch", bytes.NewBuffer(shortenBatchReqJSON))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	response, err = client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	var shortenBatchResp []ShortenBatchResp
	body, err = io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	response.Body.Close()
	err = json.Unmarshal(body, &shortenBatchResp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Shortened URLs:", shortenBatchResp)
}

func getURLID(result string) string {
	idx := strings.LastIndexByte(result, '/')
	return result[idx+1:]
}
