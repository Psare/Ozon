package models

import "time"

type Link struct {
    OriginalURL string
    ShortURL    string
    CreatedAt   time.Time
    ExpiresAt   time.Time
}

type Storage interface {
	SaveLinkToDB(link *Link) error
	GetLinkFromDB(originalURL string, i int) (*Link, error)
	CleanupExpiredLinks()
}

type Result struct {
	Status    string
	Link      string
	ShortURL  string
	CleanupIn time.Duration
}