package main

import (
	"net/http"

	"github.com/pressly/chi"
	"github.com/rs/cors"
)

func main() {
	r := chi.NewRouter()

	// Use default options
	r.Use(cors.Default().Handler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	http.ListenAndServe(":8080", r)
}
