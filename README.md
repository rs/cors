# Go CORS handler

CORS is a `net/http` handler implementing Cross Origin Resource Sharing W3 specification (http://www.w3.org/TR/cors/) in Golang.

## Getting Started

After installing Go and setting up your [GOPATH](http://golang.org/doc/code.html#GOPATH), create your first `.go` file. We'll call it `server.go`.

```go
package main

import (
    "github.com/rs/cors"
    "net/http"
)

func main() {
    c := cors.New(cors.Options{
        AllowedOrigins: []string{"foo.com"},
    })

    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte("{\"hello\": \"world\"}"))
    })

    mux := http.NewServeMux()
    mux.Handle("/", c.Handler(h))
    http.ListenAndServe(":8080", mux)
}
```

Install `cors`:

    go get github.com/rs/cors

Then run your server:

    go run server.go

The server now runs on `localhost:8080`:

    $ curl -D - -H 'Origin: foo.com' http://localhost:8080/
    HTTP/1.1 200 OK
    Access-Control-Allow-Origin: foo.com
    Content-Type: application/json
    Date: Sat, 25 Oct 2014 03:43:57 GMT
    Content-Length: 18

    {"hello": "world"}

or:

    $ curl -D - -H 'Origin: bar.com' http://localhost:8080/
    HTTP/1.1 200 OK
    Content-Type: application/json
    Date: Sat, 25 Oct 2014 03:44:25 GMT
    Content-Length: 18

    {"hello": "world"}

See [API documentation](http://godoc.org/github.com/rs/cors) for more info.

## Licenses

All source code is licensed under the [MIT License](https://raw.github.com/rs/cors/master/LICENSE).
