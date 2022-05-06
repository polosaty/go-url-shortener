package storage

import (
	"fmt"
	"log"
	"sync"
)

type MemoryMap struct {
	Mutex      sync.RWMutex
	urls       map[URL]URL
	UserShorts map[string]map[URL]struct{}
}

func (d *MemoryMap) Ping() bool {
	return true
}

func NewMemoryMap() *MemoryMap {
	db := &MemoryMap{
		urls:       make(map[URL]URL),
		UserShorts: make(map[string]map[URL]struct{}),
	}
	return db
}

func (d *MemoryMap) SaveLongURL(long URL, userID string) (URL, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	shortURL, err := makeShort(long)
	if err != nil {
		return "", fmt.Errorf("cannot generate short url: %w", err)
	}
	d.SetLongURL(long, shortURL, userID)
	return shortURL, nil
}

func (d *MemoryMap) SetLongURL(long URL, short URL, userID string) {
	d.urls[short] = long
	userShorts, exists := d.UserShorts[userID]
	if !exists {
		userShorts = make(map[URL]struct{})
		d.UserShorts[userID] = userShorts
	}
	userShorts[short] = struct{}{}
}

func (d *MemoryMap) GetLongURL(short URL) (URL, error) {
	d.Mutex.RLock()
	defer d.Mutex.RUnlock()

	longURL := d.urls[short]
	if longURL == "" {
		return longURL, fmt.Errorf("short url not registered")
	}

	return longURL, nil
}

func (d *MemoryMap) GetUsersURLs(userID string) (result []URLPair) {
	for short := range d.UserShorts[userID] {
		result = append(result, URLPair{
			ShortURL: short,
			LongURL:  d.urls[short],
		})
	}
	return
}

func (d *MemoryMap) SaveLongBatchURL(longURLS []CorrelationLongPair, userID string) ([]CorrelationShortPair, error) {
	result := make([]CorrelationShortPair, 0, len(longURLS))
	for _, p := range longURLS {

		short, err := d.SaveLongURL(p.LongURL, userID)
		if err != nil {
			log.Printf("SaveLongBatchURL error(%v):  cant shor url %v", err, p.LongURL)
			continue
		}
		result = append(result, CorrelationShortPair{p.CorrelationID, short})
	}
	return result, nil
}
