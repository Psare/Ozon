package rout

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"ozon/database"
	"ozon/handlers"
	"ozon/inmemory"
	"ozon/routinmem"
	"ozon/apimem"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	letterBytes      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	shortURLLength   = 10
	ShortURLBaseURL  = "http://localhost:8000/" 
	dbFileName       = "project.db"              
	dbMaxIdleConns   = 10
	dbMaxOpenConns   = 100
	cleanupInterval  = time.Hour * 24 * 30          
	maxURLExpiration = time.Hour * 24 * 7	
)

var (
	dbMutex   sync.Mutex
	dbPool    *sql.DB
	templates *template.Template
)

type Result struct {
	Status    string
	Link      string
	ShortURL  string
	CleanupIn time.Duration
}

func getCleanupTime(created time.Time, expires time.Time) time.Duration {
	return expires.Sub(time.Now())
}

func FormatCleanupTime(duration time.Duration) string {
	days := duration / (time.Hour * 24)
	hours := (duration % (time.Hour * 24)) / time.Hour
	minutes := (duration % time.Hour) / time.Minute
	seconds := (duration % time.Minute) / time.Second
	return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Println(err)
	}

	funcMap := template.FuncMap{
		"formatCleanupTime": FormatCleanupTime,
	}

	templ = templ.Funcs(funcMap)

	if r.Method == "GET" {
		err = templ.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	} else if r.Method == "POST" {
		result := Result{}
		if !database.IsValidURL(r.FormValue("url")) {
			result.Status = "Invalid URL format!"
		} else {
			result.Link = r.FormValue("url")

			db, err := sql.Open("postgres", "postgres://myuser:1234@localhost/ozon?sslmode=disable")
			if err != nil {
				log.Println(err)
			}
			defer db.Close()

			_, err = db.Exec(`CREATE TABLE IF NOT EXISTS links (
				link TEXT,
				short TEXT,
				created_at TIMESTAMPTZ,
				expires_at TIMESTAMPTZ
			)`)
			if err != nil {
				log.Println(err)
			}

			var shortURL string
			err = db.QueryRow("SELECT short FROM links WHERE link=$1", result.Link).Scan(&shortURL)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Println(err)
				}
				shortURL = database.GenerateShortURL()
				_, err = db.Exec("INSERT INTO links (link, short, created_at, expires_at) VALUES ($1, $2, CURRENT_TIMESTAMP, $3)", result.Link, shortURL, time.Now().Add(maxURLExpiration))
				if err != nil {
					log.Println(err)
				}
			} else {
				var createdAt time.Time
				var expiresAt time.Time
				err = db.QueryRow("SELECT created_at, expires_at FROM links WHERE link=$1", result.Link).Scan(&createdAt, &expiresAt)
				if err != nil {
					log.Println(err)
				}
				if expiresAt.Before(time.Now()) {
					_, err := db.Exec("DELETE FROM links WHERE link=$1", result.Link)
					if err != nil {
						log.Println(err)
					}
					result.Status = "Link has expired and has been deleted"
				}
			}

			result.ShortURL = fmt.Sprintf("%s%s", ShortURLBaseURL, shortURL)
			result.Status = "Shortening successful"
			result.CleanupIn = getCleanupTime(time.Now(), time.Now().Add(maxURLExpiration))
		}

		err = templ.Execute(w, result)
		if err != nil {
			log.Println(err)
		}
	}
}

func redirectTo(w http.ResponseWriter, r *http.Request) {
	var link string
	vars := mux.Vars(r)

	db, err := sql.Open("postgres", "postgres://myuser:1234@localhost/ozon?sslmode=disable")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT link FROM links WHERE short=$1 AND expires_at > CURRENT_TIMESTAMP LIMIT 1", vars["key"]).Scan(&link)
	if err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, link, http.StatusFound)
}

func Postmain() {
	database.InitDB()
	templates = template.Must(template.ParseFiles("templates/index.html"))

	go func() {
		for {
			time.Sleep(cleanupInterval)
			database.CleanupExpiredLinks()
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/", indexPage)
	router.HandleFunc("/shorten", indexPage)
	router.HandleFunc("/{key}", redirectTo)

	// API endpoints
	router.HandleFunc("/api/shorten", handlers.ApiShortenURL).Methods("POST")
	router.HandleFunc("/api/{key}", handlers.ApiGetURL).Methods("GET")

	log.Println(http.ListenAndServe(":8000", router))
}

func PostmainInMemory() {
	storage := inmemory.NewInMemoryStorage(cleanupInterval)
	storage.StartCleanupRoutine()
	templates = template.Must(template.ParseFiles("templates/index.html"))
	router := mux.NewRouter()
	router.HandleFunc("/", routinmem.IndexPageMem)
	router.HandleFunc("/shorten", routinmem.IndexPageMem)
	router.HandleFunc("/{key}", routinmem.RedirectToMem)

	// API endpoints
	router.HandleFunc("/api/shorten", apimem.ApiShortenURL).Methods("POST")
	router.HandleFunc("/api/{key}", apimem.ApiGetURL).Methods("GET")

	log.Println(http.ListenAndServe(":8000", router))
}
