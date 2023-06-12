package database

import (
	"database/sql"
	"log"
	"math/rand"
	"net/url"
	"ozon/models"
	"sync"
	"time"
	"os"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	letterBytes      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	shortURLLength   = 10
	dbMaxIdleConns   = 10
	dbMaxOpenConns   = 100
	cleanupInterval  = time.Hour * 24 * 30
	maxURLExpiration = time.Hour * 24 * 7
)

var (
	dbMutex sync.Mutex
	dbPool  *sql.DB
)


func InitDB() {
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxIdleConns(dbMaxIdleConns)
	db.SetMaxOpenConns(dbMaxOpenConns)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS links (
		link TEXT PRIMARY KEY,
		short TEXT,
		created_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ
	)`)
	if err != nil {
		log.Fatal(err)
	}

	dbPool = db
}

func GenerateShortURL() string {
	rand.Seed(time.Now().UnixNano())

	shortURL := make([]byte, shortURLLength)
	for i := 0; i < shortURLLength; i++ {
		shortURL[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(shortURL)
}

func IsValidURL(token string) bool {
	_, err := url.ParseRequestURI(token)
	if err != nil {
		return false
	}
	u, err := url.Parse(token)
	if err != nil || u.Host == "" {
		return false
	}
	return true
}

func SaveLinkToDB(link *models.Link) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	_, err := dbPool.Exec("INSERT INTO links (link, short, created_at, expires_at) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
		link.OriginalURL, link.ShortURL, link.CreatedAt, link.ExpiresAt)
	if err != nil {
		return err
	}

	return nil
}

func CleanupExpiredLinks() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	_, err := dbPool.Exec("DELETE FROM links WHERE expires_at < $1", time.Now())
	if err != nil {
		log.Println("Error cleaning up expired links:", err)
	}
}

func GetLinkFromDB(originalURL string, i int) (*models.Link, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	var link models.Link
	var query string

	if i == 0 {
		query = "SELECT link, short, created_at, expires_at FROM links WHERE link = $1"
	} else if i == 1 {
		query = "SELECT link, short, created_at, expires_at FROM links WHERE short = $1"
	} else {
		return nil, nil
	}

	err := dbPool.QueryRow(query, originalURL).Scan(
		&link.OriginalURL, &link.ShortURL, &link.CreatedAt, &link.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
			}
		return nil, err
	}

	return &link, nil
}
