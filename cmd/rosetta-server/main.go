package main

import (
	"log"
	"net/http"
	"os"

	"rosetta/internal/service"
)

func main() {
	addr := os.Getenv("ROSETTA_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, service.NewHandler()); err != nil {
		log.Fatal(err)
	}
}
