package main

import (
	"log"
	"net/http"

	"github.com/dengbin9009/DePu/backend/internal/api"
)

func main() {
	server := api.NewServer()
	log.Println("DePu API listening on :8080")
	if err := http.ListenAndServe(":8080", server.Routes()); err != nil {
		log.Fatal(err)
	}
}
