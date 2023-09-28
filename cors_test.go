package cors

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

var testResponse = []byte("bar")
var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(testResponse)
})

var allHeaders = []string{
	"Vary",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
	"Access-Control-Allow-Private-Network",
	"Access-Control-Max-Age",
	"Access-Control-Expose-Headers",
}

func assertHeaders(t *testing.T, resHeaders http.Header, expHeaders map[string]string) {
	t.Helper()
	for _, name := range allHeaders {
		got := strings.Join(resHeaders[name], ", ")
		want := expHeaders[name]
		if got != want {
			t.Errorf("Response header %q = %q, want %q", name, got, want)
		}
	}
}

func assertResponse(t *testing.T, res *httptest.ResponseRecorder, responseCode int) {
	t.Helper()
	if responseCode != res.Code {
		t.Errorf("assertResponse: expected response code to be %d but got %d. ", responseCode, res.Code)
	}
}

func TestSpec(t *testing.T) {
	cases := []struct {
		name          string
		options       Options
		method        string
		reqHeaders    map[string]string
		resHeaders    map[string]string
		originAllowed bool
	}{
		{
			"NoConfig",
			Options{
				// Intentionally left blank.
			},
			"GET",
			map[string]string{},
			map[string]string{
				"Vary": "Origin",
			},
			true,
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
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "*",
			},
			true,
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
			map[string]string{
				"Vary":                             "Origin",
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
			},
			true,
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
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
			true,
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
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foo.bar.com",
			},
			true,
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
			map[string]string{
				"Vary": "Origin",
			},
			false,
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
			map[string]string{
				"Vary": "Origin",
			},
			false,
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
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
			true,
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
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
			true,
		},
		{
			"AllowOriginVaryRequestFuncMatch",
			Options{
				AllowOriginVaryRequestFunc: func(r *http.Request, o string) (bool, []string) {
					return regexp.MustCompile("^http://foo").MatchString(o) && r.Header.Get("Authorization") == "secret", []string{"Authorization"}
				},
			},
			"GET",
			map[string]string{
				"Origin":        "http://foobar.com",
				"Authorization": "secret",
			},
			map[string]string{
				"Vary":                        "Origin, Authorization",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
			true,
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
			map[string]string{
				"Vary": "Origin",
			},
			false,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://example.com/",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Max-Age":       "10",
			},
			true,
		},
		{
			"MaxAgeNegative",
			Options{
				AllowedOrigins: []string{"http://example.com/"},
				AllowedMethods: []string{"GET"},
				MaxAge:         -1,
			},
			"OPTIONS",
			map[string]string{
				"Origin":                        "http://example.com/",
				"Access-Control-Request-Method": "GET",
			},
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://example.com/",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Max-Age":       "0",
			},
			true,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "PUT",
			},
			true,
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
			map[string]string{
				"Vary": "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
			},
			true,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Header-2, X-Header-1",
			},
			true,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Requested-With",
			},
			true,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
				"Access-Control-Allow-Headers": "X-Header-2, X-Header-1",
			},
			true,
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
			map[string]string{
				"Vary": "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
			},
			true,
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
			map[string]string{
				"Vary":                          "Origin",
				"Access-Control-Allow-Origin":   "http://foobar.com",
				"Access-Control-Expose-Headers": "X-Header-1, X-Header-2",
			},
			true,
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
			map[string]string{
				"Vary":                             "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":      "http://foobar.com",
				"Access-Control-Allow-Methods":     "GET",
				"Access-Control-Allow-Credentials": "true",
			},
			true,
		},
		{
			"AllowedPrivateNetwork",
			Options{
				AllowedOrigins:      []string{"http://foobar.com"},
				AllowPrivateNetwork: true,
			},
			"OPTIONS",
			map[string]string{
				"Origin":                                 "http://foobar.com",
				"Access-Control-Request-Method":          "GET",
				"Access-Control-Request-Private-Network": "true",
			},
			map[string]string{
				"Vary":                                 "Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Request-Private-Network",
				"Access-Control-Allow-Origin":          "http://foobar.com",
				"Access-Control-Allow-Methods":         "GET",
				"Access-Control-Allow-Private-Network": "true",
			},
			true,
		},
		{
			"DisallowedPrivateNetwork",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			map[string]string{
				"Origin":                                "http://foobar.com",
				"Access-Control-Request-Method":         "GET",
				"Access-Control-Request-PrivateNetwork": "true",
			},
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "http://foobar.com",
				"Access-Control-Allow-Methods": "GET",
			},
			true,
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
			map[string]string{
				"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
			true,
		},
		{
			"NonPreflightOptions",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			map[string]string{
				"Origin": "http://foobar.com",
			},
			map[string]string{
				"Vary":                        "Origin",
				"Access-Control-Allow-Origin": "http://foobar.com",
			},
			true,
		},
	}
	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			s := New(tc.options)

			req, _ := http.NewRequest(tc.method, "http://example.com/foo", nil)
			for name, value := range tc.reqHeaders {
				req.Header.Add(name, value)
			}

			t.Run("OriginAllowed", func(t *testing.T) {
				if have, want := s.OriginAllowed(req), tc.originAllowed; have != want {
					t.Errorf("OriginAllowed have: %t want: %t", have, want)
				}
			})

			t.Run("Handler", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.Handler(testHandler).ServeHTTP(res, req)
				assertHeaders(t, res.Header(), tc.resHeaders)
			})
			t.Run("HandlerFunc", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.HandlerFunc(res, req)
				assertHeaders(t, res.Header(), tc.resHeaders)
			})
			t.Run("Negroni", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.ServeHTTP(res, req, testHandler)
				assertHeaders(t, res.Header(), tc.resHeaders)
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

type testLogger struct {
	buf *bytes.Buffer
}

func (l *testLogger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.buf, format, v...)
}

func TestLogger(t *testing.T) {
	logger := &testLogger{buf: &bytes.Buffer{}}
	s := New(Options{
		Logger: logger,
	})

	if s.Log == nil {
		t.Error("Logger not created when Logger is set")
	}
	s.logf("test")
	if logger.buf.String() != "test" {
		t.Error("Logger not used")
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

func TestOptionsSuccessStatusCodeDefault(t *testing.T) {
	s := New(Options{
		// Intentionally left blank.
	})

	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Access-Control-Request-Method", "GET")

	t.Run("Handler", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.Handler(testHandler).ServeHTTP(res, req)
		assertResponse(t, res, http.StatusNoContent)
	})
	t.Run("HandlerFunc", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.HandlerFunc(res, req)
		assertResponse(t, res, http.StatusNoContent)
	})
	t.Run("Negroni", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.ServeHTTP(res, req, testHandler)
		assertResponse(t, res, http.StatusNoContent)
	})
}

func TestOptionsSuccessStatusCodeOverride(t *testing.T) {
	s := New(Options{
		OptionsSuccessStatus: http.StatusOK,
	})

	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Access-Control-Request-Method", "GET")

	t.Run("Handler", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.Handler(testHandler).ServeHTTP(res, req)
		assertResponse(t, res, http.StatusOK)
	})
	t.Run("HandlerFunc", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.HandlerFunc(res, req)
		assertResponse(t, res, http.StatusOK)
	})
	t.Run("Negroni", func(t *testing.T) {
		res := httptest.NewRecorder()
		s.ServeHTTP(res, req, testHandler)
		assertResponse(t, res, http.StatusOK)
	})
}

func TestCorsAreHeadersAllowed(t *testing.T) {
	cases := []struct {
		name             string
		allowedHeaders   []string
		requestedHeaders []string
		want             bool
	}{
		{
			name:             "nil allowedHeaders",
			allowedHeaders:   nil,
			requestedHeaders: []string{"X-PINGOTHER, Content-Type"},
			want:             false,
		},
		{
			name:             "star allowedHeaders",
			allowedHeaders:   []string{"*"},
			requestedHeaders: []string{"X-PINGOTHER, Content-Type"},
			want:             true,
		},
		{
			name:             "empty reqHeader",
			allowedHeaders:   nil,
			requestedHeaders: []string{},
			want:             true,
		},
		{
			name:             "match allowedHeaders",
			allowedHeaders:   []string{"Content-Type", "X-PINGOTHER", "X-APP-KEY"},
			requestedHeaders: []string{"X-PINGOTHER, Content-Type"},
			want:             true,
		},
		{
			name:             "not matched allowedHeaders",
			allowedHeaders:   []string{"X-PINGOTHER"},
			requestedHeaders: []string{"X-API-KEY, Content-Type"},
			want:             false,
		},
		{
			name:             "allowedHeaders should be a superset of requestedHeaders",
			allowedHeaders:   []string{"X-PINGOTHER"},
			requestedHeaders: []string{"X-PINGOTHER, Content-Type"},
			want:             false,
		},
	}

	for _, tt := range cases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			c := New(Options{AllowedHeaders: tt.allowedHeaders})
			have := c.areHeadersAllowed(convert(splitHeaderValues(tt.requestedHeaders), http.CanonicalHeaderKey))
			if have != tt.want {
				t.Errorf("Cors.areHeadersAllowed() have: %t want: %t", have, tt.want)
			}
		})
	}
}

func TestAccessControlExposeHeadersPresence(t *testing.T) {
	cases := []struct {
		name    string
		options Options
		want    bool
	}{
		{
			name:    "omit",
			options: Options{},
			want:    false,
		},
		{
			name: "include",
			options: Options{
				ExposedHeaders: []string{"X-Something"},
			},
			want: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.options)

			req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
			req.Header.Add("Origin", "http://foobar.com")

			assertExposeHeaders := func(t *testing.T, resHeaders http.Header) {
				if _, have := resHeaders["Access-Control-Expose-Headers"]; have != tt.want {
					t.Errorf("Access-Control-Expose-Headers have: %t want: %t", have, tt.want)
				}
			}

			t.Run("Handler", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.Handler(testHandler).ServeHTTP(res, req)
				assertExposeHeaders(t, res.Header())
			})
			t.Run("HandlerFunc", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.HandlerFunc(res, req)
				assertExposeHeaders(t, res.Header())
			})
			t.Run("Negroni", func(t *testing.T) {
				res := httptest.NewRecorder()
				s.ServeHTTP(res, req, testHandler)
				assertExposeHeaders(t, res.Header())
			})
		})
	}

}
