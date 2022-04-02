package handlers

import (
	"encoding/json"
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
	r.Post("/api/shorten", r.PostLongGetShortJSON())
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
		http.Redirect(w, r, long.S(), http.StatusTemporaryRedirect)
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
		longStr := storage.URL(long)
		shortURL, err := h.Repository.SaveLongURL(longStr)
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

type PostLongJSONRequest struct {
	URL storage.URL `json:"url"`
}

type PostLongJSONResponse struct {
	Result storage.URL `json:"result"`
}

func (h *MainHandler) PostLongGetShortJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestJSON PostLongJSONRequest
		var responseJSON PostLongJSONResponse

		if err := json.NewDecoder(r.Body).Decode(&requestJSON); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		shortURL, err := h.Repository.SaveLongURL(requestJSON.URL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("cant make short url", err)
			return
		}
		responseJSON.Result = storage.URL(h.Location) + shortURL
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(responseJSON)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("write answer error", err)
			return
		}
	}
}
