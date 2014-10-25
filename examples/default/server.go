package main

import (
	"github.com/rs/cors"
	"net/http"
)

func main() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"hello\": \"world\"}"))
	})

	mux := http.NewServeMux()
	// Use default options
	mux.Handle("/", cors.Default().Handler(h))
	http.ListenAndServe(":8080", mux)
}
