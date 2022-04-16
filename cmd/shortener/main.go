package main

import (
	"flag"
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

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base url for short urls")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file for save/load urls")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database DSN")
	flag.Parse()

	var db storage.Repository

	if cfg.DatabaseDSN != "" {
		if db, err = storage.NewPG(cfg.DatabaseDSN); err != nil {
			log.Fatal(err)
		}
		log.Println("use postgres conn " + cfg.DatabaseDSN + " as db")
	} else if cfg.FileStoragePath != "" {
		if db, err = storage.NewFileStorage(cfg.FileStoragePath); err != nil {
			log.Fatal(err)
		}
		log.Println("use file " + cfg.FileStoragePath + " as db")
	} else {
		db = storage.NewMemoryMap()
	}

	log.Fatal(server.Serve(cfg.ServerAddress, cfg.BaseURL, db))
}
