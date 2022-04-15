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
	Repository storage.Repository
	Location   string
}

func NewMainHandler(repository storage.Repository, location string) *MainHandler {

	var secretKey = []byte("secret key") // TODO: make random and save

	h := &MainHandler{Mux: chi.NewMux(), Repository: repository, Location: location}
	h.Use(gzipInput)
	h.Use(gzipOutput)
	h.Use(middleware.RequestID)
	h.Use(middleware.RealIP)
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	h.Use(authMiddleware(secretKey))
	h.Post("/", h.PostLongGetShort())
	h.Route("/api", func(r chi.Router) {
		r.Post("/shorten", h.PostLongGetShortJSON())
		r.Get("/user/urls", h.GetUserUrlsJSON())
	})

	h.Get("/{short}", h.GetLong())

	return h
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
		http.Redirect(w, r, long.S(), http.StatusTemporaryRedirect)
	}
}

func (h *MainHandler) PostLongGetShort() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		long, err := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("string read error", err)
			return
		}
		longStr := storage.URL(long)
		shortURL, err := h.Repository.SaveLongURL(longStr, session.UserID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("cant make short url", err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprint(w, h.Location+shortURL.S())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("write answer error", err)
			return
		}
	}
}
