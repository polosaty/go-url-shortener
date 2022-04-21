package handlers

import (
	"encoding/json"
	"go-url-shortener/internal/app/storage"
	"log"
	"net/http"
)

type GetUserUrlsJSONResponse []storage.URLPair

func (h *MainHandler) GetUserUrlsJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var responseJSON GetUserUrlsJSONResponse
		session := GetSession(r)
		responseJSON = h.Repository.GetUsersUrls(session.UserID)
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

		shortURL, err := h.Repository.SaveLongURL(requestJSON.URL, session.UserID)
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
