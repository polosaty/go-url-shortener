package server

import (
	"go-url-shortener/internal/app/handlers"
	"go-url-shortener/internal/app/storage"
	"net/http"
	"sync"
)

func Serve(addr string) error {
	var mutex sync.Mutex
	db := storage.DB{
		Urls:  make(map[storage.URL]storage.URL),
		Mutex: &mutex,
	}

	handler := handlers.MainHandler{
		Repository: &db,
		Location:   "http://" + addr + "/",
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server.ListenAndServe()
}
