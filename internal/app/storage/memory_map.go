package storage

import (
	"fmt"
	"strconv"
	"sync"
)

type MemoryMap struct {
	Mutex *sync.RWMutex
	Urls  map[URL]URL
}

func NewMemoryMap() *MemoryMap {
	db := &MemoryMap{
		Urls:  make(map[URL]URL),
		Mutex: &sync.RWMutex{},
	}
	return db
}

func (d *MemoryMap) SaveLongURL(long URL) (URL, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	short, err := Hash(long.S())
	if err != nil {
		return "", fmt.Errorf("cant make short url: %w", err)
	}
	shortURL := URL(strconv.FormatUint(uint64(short), 16))
	d.Urls[shortURL] = long
	return shortURL, nil
}

func (d *MemoryMap) GetLongURL(short URL) (URL, error) {
	d.Mutex.RLock()
	defer d.Mutex.RUnlock()

	longURL := d.Urls[short]
	if longURL == "" {
		return longURL, fmt.Errorf("short url not registered")
	}

	return longURL, nil
}
