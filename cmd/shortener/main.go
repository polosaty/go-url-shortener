package main

import (
	"github.com/caarlos0/env/v6"
	"go-url-shortener/internal/app/config"
	"go-url-shortener/internal/app/server"
	"log"
)

func main() {
	var cfg config.Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Serve(cfg.ServerAddress, cfg.BaseURL))
}
