# Go CORS handler

CORS is a `net/http` handler implementing [Cross Origin Resource Sharing W3 specification](http://www.w3.org/TR/cors/) in Golang.

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

### More Examples

* `net/http`: [examples/nethttp/server.go](https://github.com/rs/cors/blob/master/examples/nethttp/server.go)
* [Goji](https://goji.io): [examples/goji/server.go](https://github.com/rs/cors/blob/master/examples/goji/server.go)
* [Martini](http://martini.codegangsta.io): [examples/martini/server.go](https://github.com/rs/cors/blob/master/examples/martini/server.go)
* [Negori](https://github.com/codegangsta/negroni): [examples/negori/server.go](https://github.com/rs/cors/blob/master/examples/negori/server.go)
* [Alice](https://github.com/justinas/alice): [examples/alice/server.go](https://github.com/rs/cors/blob/master/examples/alice/server.go)

## Parameters

Parameters are passed to the middleware thru the `cors.New` method as follow:

```go
c := cors.New(cors.Options{
    AllowedOrigins: []string{"foo.com"},
    AllowCredentials: true,
})
```

* **AllowedOrigins** `[]string`: A list of domains a cross-domain request can be executed from. If the special `*` value is present in the list, all domains will be allowed. The default value is `*`.
* **AllowedMethods** `[]string`: A list of methods the client is allowed to use with cross-domain requests.
* **AllowedHeaders** `[]string`: A list of non simple headers the client is allowed to use with cross-domain requests. Default value is simple methods (`GET` and `POST`)
* **ExposedHeaders** `[]string`: Indicates which headers are safe to expose to the API of a CORS API specification
* **AllowCredentials** `bool`: Indicates whether the request can include user credentials like cookies, HTTP authentication or client side SSL certificates. The default is `false`.
* **MaxAge** `int`: Indicates how long (in seconds) the results of a preflight request can be cached. The default is `0` which stands for no max age.

See [API documentation](http://godoc.org/github.com/rs/cors) for more info.

## Licenses

All source code is licensed under the [MIT License](https://raw.github.com/rs/cors/master/LICENSE).
