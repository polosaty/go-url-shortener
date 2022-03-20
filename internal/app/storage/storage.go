package storage

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"
)

type URL string

func (u *URL) S() string {
	return string(*u)
}

type Repository interface {
	SaveLongURL(long URL) (URL, error)
	GetLongURL(short URL) (URL, error)
}

type DB struct {
	Mutex *sync.Mutex
	Urls  map[URL]URL
}

func (d *DB) SaveLongURL(long URL) (URL, error) {
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

func (d *DB) GetLongURL(short URL) (URL, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	longURL := d.Urls[short]
	if longURL == "" {
		return longURL, fmt.Errorf("short url not registered")
	}

	return longURL, nil
}

func Hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	return h.Sum32(), err
}
