package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dengbin9009/DePu/backend/internal/api"
)

func main() {
	addr := os.Getenv("DEPU_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := api.NewServer()
	log.Println("DePu API listening on " + addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
