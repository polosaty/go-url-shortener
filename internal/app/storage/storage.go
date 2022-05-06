package storage

import (
	"errors"
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
	GetUsersURLs(userID string) []URLPair
	DeleteUsersURLs(userID string, shortUrls ...URL) error
	DelayedDeleteUsersURLs(userID string, shortUrls ...URL) error
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

var ErrConflictURL = errors.New("url already exists")
var ErrDeletedURL = errors.New("url deleted")

type ConflictURLError struct {
	Err      error
	ShortURL URL
}

// Error добавляет поддержку интерфейса error для типа ConflictURLError.
func (e *ConflictURLError) Error() string {
	return fmt.Sprintf("[%v] %v", (e.ShortURL), e.Err)
}

// NewConflictURLError упаковывает ошибку err в тип ConflictURLError c текущим временем.
func NewConflictURLError(shortURL URL, err error) error {
	return &ConflictURLError{
		ShortURL: shortURL,
		Err:      err,
	}
}

// Unwrap добавляет возможность распаковывать оригинальную ошибку
func (e *ConflictURLError) Unwrap() error {
	return e.Err
}
