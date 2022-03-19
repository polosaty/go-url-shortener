package handlers

import (
	"fmt"
	"go-url-shortener/internal/app/storage"
	"io"
	"log"
	"net/http"
)

type MainHandler struct {
	Repository storage.Repository
	Location   string
}

func (h MainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

		long, err := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("string read error", err)
			return
		}
		longStr := string(long)
		shortURL, err := h.Repository.SaveLongURL(storage.URL(longStr))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("cant make short url", err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprint(w, h.Location+string(shortURL))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("write answer error", err)
			return
		}
	} else if r.Method == http.MethodGet {
		short := r.URL.Path[1:]
		long, err := h.Repository.GetLongURL(storage.URL(short))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, string(long), http.StatusTemporaryRedirect)
	}
}
