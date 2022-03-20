package main

import (
	"go-url-shortener/internal/app/server"
	"log"
)

func main() {
	log.Fatal(server.Serve("localhost:8080"))
}
