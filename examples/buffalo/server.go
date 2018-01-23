package main

import (
	"log"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/rs/cors"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{})
}

func main() {
	app := App()
	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}

func App() *buffalo.App {
	app := buffalo.New(buffalo.Options{
		PreWares: []buffalo.PreWare{cors.Default().Handler},
	})

	app.GET("/", HomeHandler)

	return app
}

func HomeHandler(c buffalo.Context) error {
	return c.Render(200, r.JSON(map[string]string{"message": "Welcome to Buffalo!"}))
}
