package routinmem

import (
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"log"
	"ozon/database"
	"ozon/inmemory"
	"ozon/models"
	"ozon/utils"
	"time"
	"fmt"
)

var maxURLExpiration = time.Hour * 24 * 7
var cleanupInterval  = time.Hour * 24 * 30
var ShortURLBaseURL  = "http://localhost:8000/"
var storage *inmemory.InMemoryStorage

func init() {
	storage = inmemory.NewInMemoryStorage(cleanupInterval)
	storage.StartCleanupRoutine()
}

func IndexPageMem(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"formatCleanupTime": utils.FormatCleanupTime,
	}

	templ = templ.Funcs(funcMap)

	if r.Method == "GET" {
		err := templ.Execute(w, nil)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		result := models.Result{}
		if !database.IsValidURL(r.FormValue("url")) {
			result.Status = "Invalid URL format!"
		} else {
			result.Link = r.FormValue("url")

			link, err := storage.GetLinkFromDB(result.Link, 1)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if link == nil {
				shortURL := database.GenerateShortURL()
				link = &models.Link{
					OriginalURL: result.Link,
					ShortURL:    shortURL,
					CreatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(maxURLExpiration),
				}

				err = storage.SaveLinkToDB(link)
				if err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				if link.ExpiresAt.Before(time.Now()) {
					err := storage.DeleteLinkFromDB(link.OriginalURL)
					if err != nil {
						log.Println(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					result.Status = "Link has expired and has been deleted"
				}
			}

			result.ShortURL = fmt.Sprintf("%s%s", ShortURLBaseURL, link.ShortURL)
			result.Status = "Shortening successful"
			result.CleanupIn = link.ExpiresAt.Sub(time.Now())
		}

		err := templ.Execute(w, result)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func RedirectToMem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortURL := vars["key"]

	link, err := storage.GetLinkFromDB(shortURL, 1)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if link == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.Redirect(w, r, link.OriginalURL, http.StatusFound)
}
