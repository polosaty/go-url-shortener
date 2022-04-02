package main

import (
	"github.com/caarlos0/env/v6"
	"go-url-shortener/internal/app/config"
	"go-url-shortener/internal/app/server"
	"go-url-shortener/internal/app/storage"
	"log"
)

func main() {
	var cfg config.Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	var db storage.Repository

	if cfg.FileStoragePath != "" {
		if db, err = storage.NewFileStorage(cfg.FileStoragePath); err != nil {
			log.Fatal(err)
		}
		log.Println("use file " + cfg.FileStoragePath + " as db")
	} else {
		db = storage.NewMemoryMap()
	}

	log.Fatal(server.Serve(cfg.ServerAddress, cfg.BaseURL, db))
}
