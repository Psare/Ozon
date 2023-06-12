package apimem

import (
	"encoding/json"
	"net/http"
	"ozon/models"
	"ozon/database"
	"ozon/inmemory"
	"fmt"
	"time"

	"github.com/gorilla/mux"
)

var maxURLExpiration = time.Hour * 24 * 7
var cleanupInterval  = time.Hour * 24 * 30
var ShortURLBaseURL  = "http://localhost:8000/"

var storage *inmemory.InMemoryStorage

func init() {
	storage = inmemory.NewInMemoryStorage(cleanupInterval)
	storage.StartCleanupRoutine()
}

func ApiGetURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type Response struct {
		OriginalURL string `json:"original_url,omitempty"`
		Error       string `json:"error,omitempty"`
	}

	vars := mux.Vars(r)
	shortURL := vars["key"]
	link, err := storage.GetLinkFromDB(shortURL, 1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: "Internal Server Error"})
		return
	}

	if link == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Error: "Link not found"})
		return
	}

	response := Response{
		OriginalURL: link.OriginalURL,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func ApiShortenURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type Request struct {
		URL string `json:"url"`
	}

	type Response struct {
		ShortURL string `json:"short_url"`
		Error    string `json:"error,omitempty"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "Invalid JSON"})
		return
	}

	if !database.IsValidURL(req.URL) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "Invalid URL format"})
		return
	}

	link, err := storage.GetLinkFromDB(req.URL, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: "Internal Server Error"})
		return
	}

	if link == nil {
		shortURL := database.GenerateShortURL()
		link = &models.Link{
			OriginalURL: req.URL,
			ShortURL:    shortURL,
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(maxURLExpiration),
		}

		if err := storage.SaveLinkToDB(link); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(Response{Error: "Internal Server Error"})
			return
		}
	}

	response := Response{
		ShortURL: fmt.Sprintf("%s%s", ShortURLBaseURL, link.ShortURL),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
