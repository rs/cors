package main

import (
	"github.com/rs/cors"
	"net/http"
)

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"hello\": \"world\"}"))
	})

	// Use default options
	handler = cors.Default().Handler(handler)
	http.ListenAndServe(":8080", handler)
}
