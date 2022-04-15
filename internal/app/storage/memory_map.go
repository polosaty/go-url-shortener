package storage

import (
	"fmt"
	"strconv"
	"sync"
)

type MemoryMap struct {
	Mutex      *sync.RWMutex
	urls       map[URL]URL
	UserShorts map[string]map[URL]struct{}
}

func NewMemoryMap() *MemoryMap {
	db := &MemoryMap{
		urls:       make(map[URL]URL),
		Mutex:      &sync.RWMutex{},
		UserShorts: make(map[string]map[URL]struct{}),
	}
	return db
}

func (d *MemoryMap) SaveLongURL(long URL, userID string) (URL, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	short, err := Hash(long.S())
	if err != nil {
		return "", fmt.Errorf("cant make short url: %w", err)
	}
	shortURL := URL(strconv.FormatUint(uint64(short), 16))
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

func (d *MemoryMap) GetUsersUrls(userID string) (result []URLPair) {
	for short := range d.UserShorts[userID] {
		result = append(result, URLPair{
			ShortURL: short,
			LongURL:  d.urls[short],
		})
	}
	return
}
