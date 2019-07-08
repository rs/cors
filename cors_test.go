package cors

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("bar"))
})

var allHeaders = []string{
	"Vary",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
	"Access-Control-Max-Age",
	"Access-Control-Expose-Headers",
}

func assertHeaders(t *testing.T, resHeaders http.Header, expHeaders map[string]string) {
	for _, name := range allHeaders {
		got := strings.Join(resHeaders[name], ", ")
		want := expHeaders[name]
		got = sortCSV(got)
		want = sortCSV(want)
		if got != want {
			t.Errorf("Response header %q = %q, want %q", name, got, want)
		}
	}
}

func sortCSV(s string) string {
	ss := strings.Split(s, ",")
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
	}
	sort.Strings(ss)
	return strings.Join(ss, ", ")
}

func assertResponse(t *testing.T, res *httptest.ResponseRecorder, responseCode int) {
	if responseCode != res.Code {
		t.Errorf("assertResponse: expected response code to be %d but got %d. ", responseCode, res.Code)
	}
}

func TestSpec(t *testing.T) {
	cases := []struct {
		name        string
		options     Options
		method      string
		reqHeaders  map[string]string
		resHeaders  map[string]string
		wantHeaders map[string]string
	}{
		{
			"NoConfig",
			Options{
				// Intentionally left blank.
			},
			"GET",
			nil,
			nil,
			map[string]string{
				"Vary": "Origin",
			},
		},
		{
			"MatchAllOrigin",
			Options{
				AllowedOrigins: []string{"*"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			nil,
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			"MatchAllOriginWithCredentials",
			Options{
				AllowedOrigins:   []string{"*"},
				AllowCredentials: true,
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			nil,
			map[string]string{
				"Vary":                             "Origin",
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			"AllowedOrigin",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			nil,
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
		},
		{
			"WildcardOrigin",
			Options{
				AllowedOrigins: []string{"http://*.bar.com"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foo.bar.com",
			},
			nil,
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foo.bar.com",
			},
		},
		{
			"DisallowedOrigin",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"GET",
			map[string]string{
				"Origin": "http://barbaz.com",
			},
			nil,
			map[string]string{
				"Vary": "Origin",
			},
		},
		{
			"DisallowedWildcardOrigin",
			Options{
				AllowedOrigins: []string{"http://*.bar.com"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foo.baz.com",
			},
			nil,
			map[string]string{
				"Vary": "Origin",
			},
		},
		{
			"AllowedOriginFuncMatch",
			Options{
				AllowOriginFunc: func(o string) bool {
					return regexp.MustCompile("^http://foo").MatchString(o)
				},
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			nil,
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
		},
		{
			"AllowOriginRequestFuncMatch",
			Options{
				AllowOriginRequestFunc: func(r *http.Request, o string) bool {
					return regexp.MustCompile("^http://foo").MatchString(o) && r.Header.Get("Authorization") == "secret"
				},
			},
			"GET",
			map[string]string{
				"Origin":        "http://foobar.com",
				"Authorization": "secret",
			},
			nil,
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
		},
		{
			"AllowOriginRequestFuncNotMatch",
			Options{
				AllowOriginRequestFunc: func(r *http.Request, o string) bool {
					return regexp.MustCompile("^http://foo").MatchString(o) && r.Header.Get("Authorization") == "secret"
				},
			},
			"GET",
			map[string]string{
				"Origin":        "http://foobar.com",
				"Authorization": "not-secret",
			},
			nil,
			map[string]string{
				"Vary": "Origin",
			},
		},
		{
			"MaxAge",
			Options{
				AllowedOrigins: []string{"http://example.com/"},
				AllowedMethods: []string{"GET"},
				MaxAge:         10,
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://example.com/",
				"Access-Control-Request-Method": "GET",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://example.com/",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Max-Age":       "10",
			},
		},
		{
			"AllowedMethod",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedMethods: []string{"PUT", "DELETE"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://foobar.com",
				"Access-Control-Request-Method": "PUT",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "PUT",
			},
		},
		{
			"DisallowedMethod",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedMethods: []string{"PUT", "DELETE"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://foobar.com",
				"Access-Control-Request-Method": "PATCH",
			},
			nil,
			map[string]string{
				"Vary": "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
			},
		},
		{
			"AllowedHeaders",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"X-Header-1", "x-header-2"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                         "http://foobar.com",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Header-2, X-HEADER-1",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Header-2, X-Header-1",
			},
		},
		{
			"DefaultAllowedHeaders",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                         "http://foobar.com",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Requested-With",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Requested-With",
			},
		},
		{
			"AllowedWildcardHeader",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"*"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                         "http://foobar.com",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Header-2, X-HEADER-1",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Header-2, X-Header-1",
			},
		},
		{
			"DisallowedHeader",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"X-Header-1", "x-header-2"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                         "http://foobar.com",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Header-3, X-Header-1",
			},
			nil,
			map[string]string{
				"Vary": "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
			},
		},
		{
			"OriginHeader",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                         "http://foobar.com",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "origin",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "Origin",
			},
		},
		{
			"ExposedHeader",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				ExposedHeaders: []string{"X-Header-1", "x-header-2"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			nil,
			map[string]string{
				"Vary":                          "Origin",
				"Access-Control-Allow-Origin":   "http://foobar.com",
				"Access-Control-Expose-Headers": "X-Header-1, X-Header-2",
			},
		},
		{
			"ExposedWildcardHeader",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				ExposedHeaders: []string{"*"},
			},
			"GET",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			map[string]string{
				"Custom-Header": "custom Value",
				"Etag":          "test etag",

				// already supported by spec
				"Content-Type": "test content type",
			},
			map[string]string{
				"Etag":                          "test etag",
				"Custom-Header":                 "Test Value",
				"Vary":                          "Origin",
				"Access-Control-Allow-Origin":   "http://foobar.com",
				"Access-Control-Expose-Headers": "Custom-Header, Vary, Etag",
			},
		},
		{
			"AllowedCredentials",
			Options{
				AllowedOrigins:   []string{"http://foobar.com"},
				AllowCredentials: true,
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://foobar.com",
				"Access-Control-Request-Method": "GET",
			},
			nil,
			map[string]string{
				"Vary":                             "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":      "http://foobar.com",
				"Access-Control-Allow-Methods":     "GET",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			"OptionPassthrough",
			Options{
				OptionsPassthrough: true,
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://foobar.com",
				"Access-Control-Request-Method": "GET",
			},
			nil,
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
		},
		{
			"NonPreflightOptions",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			nil,
			nil,
			nil,
		},
	}
	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tc.resHeaders {
					w.Header().Set(k, v)
				}
				w.Write([]byte("bar"))
			})
			s := New(tc.options)

			req, _ := http.NewRequest(tc.method, "http://example.com/foo", nil)
			for name, value := range tc.reqHeaders {
				req.Header.Add(name, value)
			}

			t.Run("Handler", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.Handler(h).ServeHTTP(res, req)
				assertHeaders(t, res.Header(), tc.wantHeaders)
			})
			t.Run("HandlerFunc", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.HandlerFunc(res, req)
				h.ServeHTTP(&ExposeAllRespWriter{ResponseWriter: res}, req)
				assertHeaders(t, res.Header(), tc.wantHeaders)
			})
			t.Run("Negroni", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.ServeHTTP(res, req, h)
				assertHeaders(t, res.Header(), tc.wantHeaders)
			})

		})
	}
}

func TestDebug(t *testing.T) {
	s := New(Options{
		Debug: true,
	})

	if s.Log == nil {
		t.Error("Logger not created when debug=true")
	}
}

func TestDefault(t *testing.T) {
	s := Default()
	if s.Log != nil {
		t.Error("c.log should be nil when Default")
	}
	if !s.allowedOriginsAll {
		t.Error("c.allowedOriginsAll should be true when Default")
	}
	if s.allowedHeaders == nil {
		t.Error("c.allowedHeaders should be nil when Default")
	}
	if s.allowedMethods == nil {
		t.Error("c.allowedMethods should be nil when Default")
	}
}

func TestHandlePreflightInvalidOriginAbortion(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com/")

	s.handlePreflight(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Vary": "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
	})
}

func TestHandlePreflightNoOptionsAbortion(t *testing.T) {
	s := New(Options{
		// Intentionally left blank.
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)

	s.handlePreflight(res, req)

	assertHeaders(t, res.Header(), map[string]string{})
}

func TestHandleActualRequestAbortsOptionsMethod(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com/")

	s.handleActualRequest(res, req)

	assertHeaders(t, res.Header(), map[string]string{})
}

func TestHandleActualRequestInvalidOriginAbortion(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com/")

	s.handleActualRequest(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Vary": "Origin",
	})
}

func TestHandleActualRequestInvalidMethodAbortion(t *testing.T) {
	s := New(Options{
		AllowedMethods:   []string{"POST"},
		AllowCredentials: true,
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com/")

	s.handleActualRequest(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Vary": "Origin",
	})
}

func TestIsMethodAllowedReturnsFalseWithNoMethods(t *testing.T) {
	s := New(Options{
		// Intentionally left blank.
	})
	s.allowedMethods = []string{}
	if s.isMethodAllowed("") {
		t.Error("IsMethodAllowed should return false when c.allowedMethods is nil.")
	}
}

func TestIsMethodAllowedReturnsTrueWithOptions(t *testing.T) {
	s := New(Options{
		// Intentionally left blank.
	})
	if !s.isMethodAllowed("OPTIONS") {
		t.Error("IsMethodAllowed should return true when c.allowedMethods is nil.")
	}
}
