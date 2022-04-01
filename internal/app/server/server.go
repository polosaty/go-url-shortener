package server

import (
	"go-url-shortener/internal/app/handlers"
	"go-url-shortener/internal/app/storage"
	"net/http"
	"sync"
)

func Serve(addr string, baseURL string) error {
	var mutex sync.Mutex
	db := &storage.DB{
		Urls:  make(map[storage.URL]storage.URL),
		Mutex: &mutex,
	}
	//проверяем не забыт ли "/" в конце BASE_URL
	if baseURL[len(baseURL)-1:] != "/" {
		baseURL = baseURL + "/"
	}
	handler := handlers.NewMainHandler(db, baseURL)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server.ListenAndServe()
}
