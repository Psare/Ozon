package inmemory

import (
	"ozon/models"
	"sync"
	"time"
)

type InMemoryStorage struct {
	links    map[string]*models.Link
	mutex    sync.Mutex
	cleanup  time.Duration
	stopChan chan struct{}
}

func NewInMemoryStorage(cleanupInterval time.Duration) *InMemoryStorage {
	return &InMemoryStorage{
		links:    make(map[string]*models.Link),
		cleanup:  cleanupInterval,
		stopChan: make(chan struct{}),
	}
}

func (s *InMemoryStorage) SaveLinkToDB(link *models.Link) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	lol, ok := s.links[link.OriginalURL]
	if ok {
		s.links[link.OriginalURL] = lol
	}

	s.links[link.OriginalURL] = link

	return nil
}

func (s *InMemoryStorage) GetLinkFromDB(shortURL string, daysAgo int) (*models.Link, error) {
	
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, link := range s.links {
		if link.ShortURL == shortURL {
			return link, nil
		}
	}

	link, ok := s.links[shortURL]
	if !ok {
		return nil, nil
	}

	// Example logic for checking link expiration
	if time.Since(link.ExpiresAt) > 0 {
		delete(s.links, shortURL)
		return nil, nil
	}

	return link, nil
}


func (s *InMemoryStorage) DeleteLinkFromDB(originalURL string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.links, originalURL)

	return nil
}

func (s *InMemoryStorage) CleanupExpiredLinks() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for url, link := range s.links {
		if link.ExpiresAt.Before(time.Now()) {
			delete(s.links, url)
		}
	}
}

func (s *InMemoryStorage) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(s.cleanup)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.CleanupExpiredLinks()
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *InMemoryStorage) StopCleanupRoutine() {
	close(s.stopChan)
}
