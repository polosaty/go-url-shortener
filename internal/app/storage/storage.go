package storage

import (
	"hash/fnv"
)

type URL string

func (u *URL) S() string {
	return string(*u)
}

type Repository interface {
	SaveLongURL(long URL) (URL, error)
	GetLongURL(short URL) (URL, error)
}

func Hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	return h.Sum32(), err
}
