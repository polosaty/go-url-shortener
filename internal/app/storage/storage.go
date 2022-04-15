package storage

import (
	"hash/fnv"
)

type URL string

func (u *URL) S() string {
	return string(*u)
}

type URLPair struct {
	ShortURL URL `json:"short_url"`
	LongURL  URL `json:"original_url"`
}

type Repository interface {
	SaveLongURL(long URL, userID string) (URL, error)
	GetLongURL(short URL) (URL, error)
	GetUsersUrls(userID string) []URLPair
}

func Hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	return h.Sum32(), err
}
