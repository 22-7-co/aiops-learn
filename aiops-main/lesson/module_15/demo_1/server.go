package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: method=%s, path=%s\n", r.Method, r.URL.Path)
	fmt.Fprintf(w, "Hello, you've requested: %s with method: %s\n", r.URL.Path, r.Method)
}

func main() {
	http.HandleFunc("/", handler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
