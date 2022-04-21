package storage

import (
	"fmt"
	"hash/fnv"
	"strconv"
)

type URL string

func (u *URL) S() string {
	return string(*u)
}

type URLPair struct {
	ShortURL URL `json:"short_url"`
	LongURL  URL `json:"original_url"`
}

type CorrelationLongPair struct {
	CorrelationID string `json:"correlation_id"`
	LongURL       URL    `json:"original_url"`
}

type CorrelationShortPair struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      URL    `json:"short_url"`
}

type Repository interface {
	SaveLongURL(long URL, userID string) (URL, error)
	SaveLongBatchURL(longURLS []CorrelationLongPair, userID string) ([]CorrelationShortPair, error)
	GetLongURL(short URL) (URL, error)
	GetUsersUrls(userID string) []URLPair
	Ping() bool
}

func Hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	return h.Sum32(), err
}

func makeShort(long URL) (URL, error) {
	short, err := Hash(long.S())
	if err != nil {
		return "", fmt.Errorf("cant make short url: %w", err)
	}
	shortURL := URL(strconv.FormatUint(uint64(short), 16))
	return shortURL, err
}
