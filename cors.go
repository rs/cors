/*
Package cors is net/http handler to handle CORS related requests
as defined by http://www.w3.org/TR/cors/

You can configure it by passing an option struct to cors.New:

    c := cors.New(cors.Options{
        AllowedOrigins: []string{"foo.com"},
        AllowedMethods: []string{"GET", "POST", "DELETE"},
        AllowCredentials: true,
    })

See Options documentation for more options.

The resulting handler is a standard net/http handler.
*/
package cors

import (
	"net/http"
	"strconv"
	"strings"
)

// Options is a struct for specifying configuration options for the Cors middleware.
type Options struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is ["*"]
	AllowedOrigins []string
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests.
	AllowedMethods []string
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET and POST)
	AllowedHeaders []string
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int
}

type Cors struct {
	options Options
}

// New creates a new Cors handler with the provided options. Options are normalized.
func New(options Options) *Cors {
	// Normalize options
	// Note: for origins and methods matching, the spec requires a case-sensitive matching.
	// As it may error prone, we chose to ignore the spec here.
	normOptions := Options{
		AllowedOrigins: convert(options.AllowedOrigins, strings.ToLower),
		AllowedMethods: convert(options.AllowedMethods, strings.ToUpper),
		// Origin is always appended as some browsers will always request
		// for this header at preflight
		AllowedHeaders:   convert(append(options.AllowedHeaders, "Origin"), toHeader),
		ExposedHeaders:   convert(options.ExposedHeaders, toHeader),
		AllowCredentials: options.AllowCredentials,
		MaxAge:           options.MaxAge,
	}
	if len(normOptions.AllowedOrigins) == 0 {
		// Default is all origins
		normOptions.AllowedOrigins = []string{"*"}
	}
	if len(normOptions.AllowedMethods) == 0 {
		// Default is simple methods
		normOptions.AllowedMethods = []string{"GET", "POST"}
	}
	return &Cors{
		options: normOptions,
	}
}

// Default creates a new Cors handler with all default options
func Default() *Cors {
	return New(Options{})
}

// Handler apply the CORS specification on the request, and add relevant CORS headers
// as necessary.
func (cors *Cors) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			cors.handlePreflight(w, r)
		} else {
			cors.handleActualRequest(w, r)
		}
		h.ServeHTTP(w, r)
	})
}

// Martini compatible handler
func (cors *Cors) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		cors.handlePreflight(w, r)
	} else {
		cors.handleActualRequest(w, r)
	}
}

// Negroni compatible interface
func (cors *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method == "OPTIONS" {
		cors.handlePreflight(w, r)
	} else {
		cors.handleActualRequest(w, r)
	}
	next(w, r)
}

// handlePreflight handles pre-flight CORS requests
func (cors *Cors) handlePreflight(w http.ResponseWriter, r *http.Request) {
	options := cors.options
	headers := w.Header()
	origin := r.Header.Get("Origin")
	if r.Method != "OPTIONS" || origin == "" || !cors.isOriginAllowed(origin) {
		return
	}
	if !cors.isMethodAllowed(r.Header.Get("Access-Control-Request-Method")) {
		return
	}
	if !cors.areHeadersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Methods", strings.Join(options.AllowedMethods, ", "))
	if len(options.AllowedHeaders) > 0 {
		headers.Set("Access-Control-Allow-Headers", strings.Join(options.AllowedHeaders, ", "))
	}
	if options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if options.MaxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(options.MaxAge))
	}
}

// handleActualRequest handles simple cross-origin requests, actual request or redirects
func (cors *Cors) handleActualRequest(w http.ResponseWriter, r *http.Request) {
	options := cors.options
	headers := w.Header()
	origin := r.Header.Get("Origin")
	if r.Method == "OPTIONS" || origin == "" || !cors.isOriginAllowed(origin) {
		return
	}
	// Note that spec does define a way to specifically disallow a simple method like GET or
	// POST. Access-Control-Allow-Methods is only used for pre-flight requests and the
	// spec doesn't instruct to check the allowed methods for simple cross-origin requests.
	// We think it's a nice feature to be able to have control on those methods though.
	if !cors.isMethodAllowed(r.Method) {
		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	if len(options.ExposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))
	}
	if options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
}

// isOriginAllowed checks if a given origin is allowed to perform cross-domain requests
// on the endpoint
func (cors *Cors) isOriginAllowed(origin string) bool {
	allowedOrigins := cors.options.AllowedOrigins
	origin = strings.ToLower(origin)
	for _, allowedOrigin := range allowedOrigins {
		switch allowedOrigin {
		case "*":
			return true
		case origin:
			return true
		}
	}
	return false
}

// isMethodAllowed checks if a given method can be used as part of a cross-domain request
// on the endpoing
func (cors *Cors) isMethodAllowed(method string) bool {
	allowedMethods := cors.options.AllowedMethods
	if len(allowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == "OPTIONS" {
		// Always allow preflight requests
		return true
	}
	for _, allowedMethod := range allowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if a given list of headers are allowed to used within
// a cross-domain request.
func (cors *Cors) areHeadersAllowed(requestedHeaders string) bool {
	if requestedHeaders == "" {
		return true
	}
	for _, header := range strings.Split(requestedHeaders, ",") {
		header = toHeader(strings.TrimSpace(header))
		found := false
		for _, allowedHeader := range cors.options.AllowedHeaders {
			if header == allowedHeader {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
