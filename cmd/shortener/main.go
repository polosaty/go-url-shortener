package main

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type DB struct {
	Mutex *sync.Mutex
	Urls  map[string]string
}

type MyHandler struct {
	DB DB
}

func hash(s string) (uint32, error) {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	return h.Sum32(), err
}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.DB.Mutex.Lock()
		defer h.DB.Mutex.Unlock()

		long, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("string read error", err)
			return
		}
		longStr := string(long)
		short, err := hash(longStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("cant make short url", err)
			return
		}
		shortStr := strconv.FormatUint(uint64(short), 16)
		h.DB.Urls[shortStr] = longStr
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprint(w, "http://localhost:8080/"+shortStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("write answer error", err)
			return
		}
	} else if r.Method == http.MethodGet {
		short := r.URL.Path[1:]
		long := h.DB.Urls[short]
		if long == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, long, http.StatusTemporaryRedirect)
	}
}

func main() {
	var mutex sync.Mutex
	handler1 := MyHandler{

		DB: DB{
			Urls:  make(map[string]string),
			Mutex: &mutex,
		},
	}

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: handler1,
	}

	log.Fatal(server.ListenAndServe())
}
