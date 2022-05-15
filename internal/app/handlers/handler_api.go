package handlers

import (
	"encoding/json"
	"errors"
	"go-url-shortener/internal/app/storage"
	"log"
	"net/http"
)

type GetUserUrlsJSONResponse []storage.URLPair

func (h *MainHandler) GetUserUrlsJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var responseJSON GetUserUrlsJSONResponse
		session := GetSession(r)
		responseJSON = h.Repository.GetUsersURLs(session.UserID)
		for indx, record := range responseJSON {
			responseJSON[indx].ShortURL = storage.URL(h.Location + record.ShortURL.S())
		}
		encoder := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if responseJSON == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		err := encoder.Encode(responseJSON)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("write answer error", err)
			return
		}
	}
}

func (h *MainHandler) DeleteUserShortUrlsJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		var shortUrls []storage.URL
		if err := json.NewDecoder(r.Body).Decode(&shortUrls); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//if err := h.Repository.DeleteUsersURLs(session.UserID, shortUrls...); err != nil {
		if err := h.Repository.DelayedDeleteUsersURLs(session.UserID, shortUrls...); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("delete urls error", err)
			return
		}
		w.WriteHeader(http.StatusAccepted)
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

		session := GetSession(r)

		if err := json.NewDecoder(r.Body).Decode(&requestJSON); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		status := http.StatusCreated
		shortURL, err := h.Repository.SaveLongURL(requestJSON.URL, session.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrConflictURL) {
				status = http.StatusConflict
				var e *storage.ConflictURLError
				if errors.As(err, &e) {
					shortURL = e.ShortURL
				}

			} else {
				w.WriteHeader(http.StatusBadRequest)
				log.Println("cant make short url", err)
				return
			}
		}
		responseJSON.Result = storage.URL(h.Location) + shortURL
		w.WriteHeader(status)
		err = json.NewEncoder(w).Encode(responseJSON)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("write answer error", err)
			return
		}
	}
}

func (h *MainHandler) PostLongGetShortBatchJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestJSON []storage.CorrelationLongPair
		var responseJSON []storage.CorrelationShortPair

		session := GetSession(r)

		if err := json.NewDecoder(r.Body).Decode(&requestJSON); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		var err error
		responseJSON, err = h.Repository.SaveLongBatchURL(requestJSON, session.UserID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("cant make short url", err)
			return
		}

		for i := range responseJSON {
			responseJSON[i].ShortURL = storage.URL(h.Location) + responseJSON[i].ShortURL
		}

		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(responseJSON)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("write answer error", err)
			return
		}
	}
}
