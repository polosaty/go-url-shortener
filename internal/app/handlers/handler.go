package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go-url-shortener/internal/app/storage"
	"io"
	"log"
	"net/http"
)

type MainHandler struct {
	*chi.Mux
	storage.Repository
	Location string
}

func NewMainHandler(repository storage.Repository, location string) *MainHandler {

	r := &MainHandler{Mux: chi.NewMux(), Repository: repository, Location: location}
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/", r.PostLongGetShort())
	r.Get("/{short}", r.GetLong())

	return r
}

func (h *MainHandler) GetLong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//short := r.URL.Path[1:]
		short := chi.URLParam(r, "short")
		long, err := h.Repository.GetLongURL(storage.URL(short))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, string(long), http.StatusTemporaryRedirect)
	}
}

func (h *MainHandler) PostLongGetShort() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}
