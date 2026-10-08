package cors

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var testResponse = []byte("bar")
var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write(testResponse)
})

// For each key-value pair of this map, the value indicates whether the key
// is a list-based field (i.e. not a singleton field);
// see https://httpwg.org/specs/rfc9110.html#abnf.extension.
var allRespHeaders = map[string]bool{
	// see https://www.rfc-editor.org/rfc/rfc9110#section-12.5.5
	"Vary": true,
	// see https://fetch.spec.whatwg.org/#http-new-header-syntax
	"Access-Control-Allow-Origin":      false,
	"Access-Control-Allow-Credentials": false,
	"Access-Control-Allow-Methods":     true,
	"Access-Control-Allow-Headers":     true,
	"Access-Control-Max-Age":           false,
	"Access-Control-Expose-Headers":    true,
	// see https://wicg.github.io/private-network-access/
	"Access-Control-Allow-Private-Network": false,
}

func assertHeaders(t *testing.T, resHeaders http.Header, expHeaders http.Header) {
	t.Helper()
	for name, listBased := range allRespHeaders {
		got := resHeaders[name]
		want := expHeaders[name]
		if !listBased && !slices.Equal(got, want) {
			t.Errorf("Response header %q = %q, want %q", name, got, want)
			continue
		}
		if listBased && !slices.Equal(normalize(got), normalize(want)) {
			t.Errorf("Response header %q = %q, want %q", name, got, want)
			continue
		}
	}
}

// normalize normalizes a list-based field value,
// preserving both empty elements and the order of elements.
func normalize(s []string) (res []string) {
	for _, v := range s {
		for _, e := range strings.Split(v, ",") {
			e = strings.Trim(e, " \t")
			res = append(res, e)
		}
	}
	return
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
		reqHeaders    http.Header
		resHeaders    http.Header
		originAllowed bool
	}{
		{
			"NoConfig",
			Options{
				// Intentionally left blank.
			},
			"GET",
			http.Header{},
			http.Header{
				"Vary": {"Origin"},
			},
			true,
		},
		{
			"MatchAllOrigin",
			Options{
				AllowedOrigins: []string{"*"},
			},
			"GET",
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"*"},
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
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                             {"Origin"},
				"Access-Control-Allow-Origin":      {"*"},
				"Access-Control-Allow-Credentials": {"true"},
			},
			true,
		},
		{
			"AllowedOrigin",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"GET",
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
			},
			true,
		},
		{
			"WildcardOrigin",
			Options{
				AllowedOrigins: []string{"http://*.bar.com"},
			},
			"GET",
			http.Header{
				"Origin": {"http://foo.bar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foo.bar.com"},
			},
			true,
		},
		{
			"DisallowedOrigin",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"GET",
			http.Header{
				"Origin": {"http://barbaz.com"},
			},
			http.Header{
				"Vary": {"Origin"},
			},
			false,
		},
		{
			"DisallowedWildcardOrigin",
			Options{
				AllowedOrigins: []string{"http://*.bar.com"},
			},
			"GET",
			http.Header{
				"Origin": {"http://foo.baz.com"},
			},
			http.Header{
				"Vary": {"Origin"},
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
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
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
			http.Header{
				"Origin":        {"http://foobar.com"},
				"Authorization": {"secret"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
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
			http.Header{
				"Origin":        {"http://foobar.com"},
				"Authorization": {"secret"},
			},
			http.Header{
				"Vary":                        {"Origin, Authorization"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
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
			http.Header{
				"Origin":        {"http://foobar.com"},
				"Authorization": {"not-secret"},
			},
			http.Header{
				"Vary": {"Origin"},
			},
			false,
		},
		{
			"MaxAge",
			Options{
				AllowedOrigins: []string{"http://example.com"},
				AllowedMethods: []string{"GET"},
				MaxAge:         10,
			},
			"OPTIONS",
			http.Header{
				"Origin":                        {"http://example.com"},
				"Access-Control-Request-Method": {"GET"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://example.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Max-Age":       {"10"},
			},
			true,
		},
		{
			"MaxAgeNegative",
			Options{
				AllowedOrigins: []string{"http://example.com"},
				AllowedMethods: []string{"GET"},
				MaxAge:         -1,
			},
			"OPTIONS",
			http.Header{
				"Origin":                        {"http://example.com"},
				"Access-Control-Request-Method": {"GET"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://example.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Max-Age":       {"0"},
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
			http.Header{
				"Origin":                        {"http://foobar.com"},
				"Access-Control-Request-Method": {"PUT"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"PUT"},
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
			http.Header{
				"Origin":                        {"http://foobar.com"},
				"Access-Control-Request-Method": {"PATCH"},
			},
			http.Header{
				"Vary": {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
			},
			true,
		},
		{
			"AllowedHeaders",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"X-Header-1", "x-header-2", "X-HEADER-3"},
			},
			"OPTIONS",
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"x-header-1,x-header-2"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Allow-Headers": {"x-header-1,x-header-2"},
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
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"x-requested-with"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Allow-Headers": {"x-requested-with"},
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
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"x-header-1,x-header-2"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Allow-Headers": {"x-header-1,x-header-2"},
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
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"x-header-1,x-header-3"},
			},
			http.Header{
				"Vary": {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
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
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                          {"Origin"},
				"Access-Control-Allow-Origin":   {"http://foobar.com"},
				"Access-Control-Expose-Headers": {"X-Header-1, X-Header-2"},
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
			http.Header{
				"Origin":                        {"http://foobar.com"},
				"Access-Control-Request-Method": {"GET"},
			},
			http.Header{
				"Vary":                             {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":      {"http://foobar.com"},
				"Access-Control-Allow-Methods":     {"GET"},
				"Access-Control-Allow-Credentials": {"true"},
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
			http.Header{
				"Origin":                                 {"http://foobar.com"},
				"Access-Control-Request-Method":          {"GET"},
				"Access-Control-Request-Private-Network": {"true"},
			},
			http.Header{
				"Vary":                                 {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Request-Private-Network"},
				"Access-Control-Allow-Origin":          {"http://foobar.com"},
				"Access-Control-Allow-Methods":         {"GET"},
				"Access-Control-Allow-Private-Network": {"true"},
			},
			true,
		},
		{
			"DisallowedPrivateNetwork",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			http.Header{
				"Origin":                                {"http://foobar.com"},
				"Access-Control-Request-Method":         {"GET"},
				"Access-Control-Request-PrivateNetwork": {"true"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
			},
			true,
		},
		{
			"OptionPassthrough",
			Options{
				OptionsPassthrough: true,
			},
			"OPTIONS",
			http.Header{
				"Origin":                        {"http://foobar.com"},
				"Access-Control-Request-Method": {"GET"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"*"},
				"Access-Control-Allow-Methods": {"GET"},
			},
			true,
		},
		{
			"NonPreflightOptions",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
			},
			"OPTIONS",
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
			},
			true,
		}, {
			"AllowedOriginsPlusAllowOriginFunc",
			Options{
				AllowedOrigins: []string{"*"},
				AllowOriginFunc: func(origin string) bool {
					return true
				},
			},
			"GET",
			http.Header{
				"Origin": {"http://foobar.com"},
			},
			http.Header{
				"Vary":                        {"Origin"},
				"Access-Control-Allow-Origin": {"http://foobar.com"},
			},
			true,
		},
		{
			"MultipleACRHHeaders",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
			"OPTIONS",
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"authorization", "content-type"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Allow-Headers": {"authorization", "content-type"},
			},
			true,
		},
		{
			"MultipleACRHHeadersWithOWSAndEmptyElements",
			Options{
				AllowedOrigins: []string{"http://foobar.com"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
			"OPTIONS",
			http.Header{
				"Origin":                         {"http://foobar.com"},
				"Access-Control-Request-Method":  {"GET"},
				"Access-Control-Request-Headers": {"authorization\t", " ", " content-type"},
			},
			http.Header{
				"Vary":                         {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
				"Access-Control-Allow-Origin":  {"http://foobar.com"},
				"Access-Control-Allow-Methods": {"GET"},
				"Access-Control-Allow-Headers": {"authorization\t", " ", " content-type"},
			},
			true,
		},
	}
	for i := range cases {
		tc := cases[i]
		t.Run(tc.name, func(t *testing.T) {
			s := New(tc.options)

			req, _ := http.NewRequest(tc.method, "http://example.com/foo", nil)
			for name, values := range tc.reqHeaders {
				for _, value := range values {
					req.Header.Add(name, value)
				}
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

func TestWildcardOriginStructure(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		origin  string
		allowed bool
	}{
		{"CommaSeparated", "https://*.example.com", "https://github.com,https://test.example.com", false},
		{"CommaSpaceSeparated", "https://*.example.com", "https://github.com, https://test.example.com", false},
		{"SpaceSeparated", "https://*.example.com", "https://github.com https://test.example.com", false},
		{"ThreeOrigins", "https://*.example.com", "https://a.example.com,https://b.example.com,https://c.example.com", false},
		{"MissingScheme", "*://example.com", "://example.com", false},
		{"MissingSeparator", "example.*", "example.com", false},
		{"MissingAuthority", "https://*", "https://", false},
		{"Userinfo", "https://*.example.com", "https://user@foo.example.com", false},
		{"Path", "https://*.example.com", "https://other.com/foo.example.com", false},
		{"Query", "https://*.example.com", "https://other.com?foo.example.com", false},
		{"Fragment", "https://*.example.com", "https://other.com#foo.example.com", false},
		{"Backslash", "https://*.example.com", `https://other.com\foo.example.com`, false},
		{"Space", "https://*.example.com", "https://foo .example.com", false},
		{"Tab", "https://*.example.com", "https://foo\t.example.com", false},
		{"CR", "https://*.example.com", "https://foo\r.example.com", false},
		{"LF", "https://*.example.com", "https://foo\n.example.com", false},
		{"FormFeed", "https://*.example.com", "https://foo\f.example.com", false},
		{"VerticalTab", "https://*.example.com", "https://foo\v.example.com", false},
		{"Host", "https://*.example.com", "https://foo.example.com", true},
		{"NestedHost", "https://*.example.com", "https://foo.bar.example.com", true},
		{"EmptySubstitution", "https://foo*.example.com", "https://foo.example.com", true},
		{"Case", "HTTPS://*.EXAMPLE.COM", "HTTPS://Foo.Example.COM", true},
		{"Port", "https://*.example.com:8443", "https://foo.example.com:8443", true},
		{"DefaultPort", "https://*.example.com:443", "https://foo.example.com:443", true},
		{"IPv4", "http://127.*:8080", "http://127.0.0.1:8080", true},
		{"IPv6", "http://[2001:db8::*]", "http://[2001:db8::1]", true},
		{"IPv6Port", "http://[::1]:*", "http://[::1]:8080", true},
		{"Punycode", "https://*.example.com", "https://xn--maraa-rta.example.com", true},
		{"TrailingDot", "https://*.example.com.", "https://foo.example.com.", true},
		{"CustomScheme", "my-app://*.example.com", "my-app://foo.example.com", true},
		{"CommaHost", "https://*.example.com", "https://foo,bar.example.com", true},
		{"Null", "n*", "null", true},
		{"WrongScheme", "https://*.example.com", "http://foo.example.com", false},
		{"WrongHost", "https://*.example.com", "https://foo.other.com", false},
		{"WrongPort", "https://*.example.com:8443", "https://foo.example.com:443", false},
		{"NoDefaultPortNormalization", "https://*.example.com", "https://foo.example.com:443", false},
		{"NoTrailingDotNormalization", "https://*.example.com", "https://foo.example.com.", false},
		{"Empty", "https://*.example.com", "", false},
	}
	for _, tc := range cases {
		for _, credentials := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/credentials=%t", tc.name, credentials), func(t *testing.T) {
				c := New(Options{AllowedOrigins: []string{tc.pattern}, AllowCredentials: credentials})
				want := http.Header{}
				if tc.allowed {
					want.Set("Access-Control-Allow-Origin", tc.origin)
					if credentials {
						want.Set("Access-Control-Allow-Credentials", "true")
					}
				}
				testOriginHandlers(t, c, tc.origin, tc.allowed, want)
			})
		}
	}
}

func testOriginHandlers(t *testing.T, c *Cors, origin string, allowed bool, want http.Header) {
	t.Helper()
	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "http://example.com/foo", nil)
			if origin != "" {
				req.Header.Set("Origin", origin)
			}
			expected := want.Clone()
			vary := "Origin"
			if method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
				vary = "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"
				if expected.Get("Access-Control-Allow-Origin") != "" {
					expected.Set("Access-Control-Allow-Methods", http.MethodGet)
				}
			}
			expected["Vary"] = append([]string{vary}, expected["Vary"]...)
			if got := c.OriginAllowed(req); got != allowed {
				t.Errorf("OriginAllowed = %t, want %t", got, allowed)
			}
			for _, entry := range []string{"Handler", "HandlerFunc", "Negroni"} {
				t.Run(entry, func(t *testing.T) {
					res := httptest.NewRecorder()
					calls := 0
					next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						testHandler(w, r)
					})
					switch entry {
					case "Handler":
						c.Handler(next).ServeHTTP(res, req)
					case "HandlerFunc":
						c.HandlerFunc(res, req)
					case "Negroni":
						c.ServeHTTP(res, req, next)
					}
					assertHeaders(t, res.Header(), expected)
					status, wantCalls := http.StatusOK, 0
					if method == http.MethodOptions {
						status = http.StatusNoContent
					} else if entry != "HandlerFunc" {
						wantCalls = 1
					}
					assertResponse(t, res, status)
					if calls != wantCalls {
						t.Errorf("next calls = %d, want %d", calls, wantCalls)
					}
					if wantCalls == 1 && !bytes.Equal(res.Body.Bytes(), testResponse) {
						t.Errorf("body = %q, want %q", res.Body.Bytes(), testResponse)
					}
					if wantCalls == 0 && res.Body.Len() != 0 {
						t.Errorf("unexpected response body: %q", res.Body.Bytes())
					}
				})
			}
		})
	}
}

func TestOriginStructureOverrides(t *testing.T) {
	const origin = "HTTPS://github.com,https://test.example.com"
	for _, exact := range []string{origin, "custom origin", "null", ""} {
		t.Run("Exact/"+exact, func(t *testing.T) {
			c := New(Options{AllowedOrigins: []string{"https://*.example.com", exact}})
			want := http.Header{}
			if exact != "" {
				want.Set("Access-Control-Allow-Origin", exact)
			}
			testOriginHandlers(t, c, exact, true, want)
		})
	}
	for _, name := range []string{"Default", "Star", "MixedStar", "Credentials", "AllowAll"} {
		t.Run(name, func(t *testing.T) {
			var c *Cors
			switch name {
			case "Default":
				c = Default()
			case "Star":
				c = New(Options{AllowedOrigins: []string{"*"}})
			case "MixedStar":
				c = New(Options{AllowedOrigins: []string{"https://*.example.com", "*"}})
			case "Credentials":
				c = New(Options{AllowedOrigins: []string{"*"}, AllowCredentials: true})
			case "AllowAll":
				c = AllowAll()
			}
			want := http.Header{"Access-Control-Allow-Origin": {"*"}}
			if name == "Credentials" {
				want.Set("Access-Control-Allow-Credentials", "true")
			}
			testOriginHandlers(t, c, origin, true, want)
			testOriginHandlers(t, c, "", true, http.Header{})
		})
	}
	for _, kind := range []string{"Origin", "Request", "VaryRequest"} {
		for _, allowed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/allowed=%t", kind, allowed), func(t *testing.T) {
				calls := 0
				callbackOrigin := origin
				options := Options{AllowedOrigins: []string{"*"}, AllowCredentials: true}
				check := func(o string) {
					t.Helper()
					calls++
					if o != callbackOrigin {
						t.Errorf("callback origin = %q, want %q", o, callbackOrigin)
					}
				}
				options.AllowOriginFunc = func(o string) bool {
					if kind != "Origin" {
						t.Error("lower-priority callback called")
					}
					check(o)
					return allowed
				}
				var lastRequest *http.Request
				if kind != "Origin" {
					options.AllowOriginRequestFunc = func(r *http.Request, o string) bool {
						if kind != "Request" {
							t.Error("lower-priority callback called")
						}
						check(o)
						lastRequest = r
						return allowed
					}
				}
				if kind == "VaryRequest" {
					options.AllowOriginVaryRequestFunc = func(r *http.Request, o string) (bool, []string) {
						check(o)
						lastRequest = r
						return allowed, []string{"authorization"}
					}
				}
				c := New(options)
				req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
				req.Header.Set("Origin", origin)
				if got := c.OriginAllowed(req); got != allowed {
					t.Errorf("OriginAllowed = %t, want %t", got, allowed)
				}
				if kind != "Origin" && lastRequest != req {
					t.Error("callback did not receive the original request")
				}
				want := http.Header{}
				if allowed {
					want.Set("Access-Control-Allow-Origin", origin)
					want.Set("Access-Control-Allow-Credentials", "true")
				}
				if kind == "VaryRequest" {
					want.Set("Vary", "Authorization")
				}
				testOriginHandlers(t, c, origin, allowed, want)
				if calls != 9 {
					t.Errorf("callback calls = %d, want 9", calls)
				}
				callbackOrigin, calls = "", 0
				want = http.Header{}
				if kind == "VaryRequest" {
					want.Set("Vary", "Authorization")
				}
				testOriginHandlers(t, c, "", allowed, want)
				if calls != 8 {
					t.Errorf("empty-origin callback calls = %d, want 8", calls)
				}
			})
		}
	}
}

func TestWildcardOriginOptions(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, passthrough := range []bool{false, true} {
			t.Run(fmt.Sprintf("allowed=%t/passthrough=%t", allowed, passthrough), func(t *testing.T) {
				c := New(Options{
					AllowedOrigins:       []string{"https://*.example.com"},
					AllowedHeaders:       []string{"X-Test"},
					ExposedHeaders:       []string{"X-Response"},
					AllowCredentials:     true,
					AllowPrivateNetwork:  true,
					MaxAge:               60,
					OptionsPassthrough:   passthrough,
					OptionsSuccessStatus: http.StatusAccepted,
				})
				for _, method := range []string{http.MethodGet, http.MethodOptions} {
					req := httptest.NewRequest(method, "http://example.com/foo", nil)
					origin := "https://foo.example.com"
					if !allowed {
						origin = "https://other.com," + origin
					}
					req.Header.Set("Origin", origin)
					want := http.Header{"Vary": {"Origin"}}
					if allowed {
						want.Set("Access-Control-Allow-Origin", origin)
						want.Set("Access-Control-Allow-Credentials", "true")
						want.Set("Access-Control-Expose-Headers", "X-Response")
					}
					status, wantCalls := http.StatusOK, 1
					if method == http.MethodOptions {
						req.Header.Set("Access-Control-Request-Method", http.MethodGet)
						req.Header.Set("Access-Control-Request-Headers", "x-test")
						req.Header.Set("Access-Control-Request-Private-Network", "true")
						want.Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Request-Private-Network")
						want.Del("Access-Control-Expose-Headers")
						if allowed {
							want.Set("Access-Control-Allow-Methods", http.MethodGet)
							want.Set("Access-Control-Allow-Headers", "x-test")
							want.Set("Access-Control-Allow-Private-Network", "true")
							want.Set("Access-Control-Max-Age", "60")
						}
						if !passthrough {
							status, wantCalls = http.StatusAccepted, 0
						}
					}
					for _, entry := range []string{"Handler", "Negroni"} {
						res, calls := httptest.NewRecorder(), 0
						next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							calls++
							testHandler(w, r)
						})
						if entry == "Handler" {
							c.Handler(next).ServeHTTP(res, req)
						} else {
							c.ServeHTTP(res, req, next)
						}
						assertHeaders(t, res.Header(), want)
						assertResponse(t, res, status)
						if calls != wantCalls {
							t.Errorf("%s %s next calls = %d, want %d", entry, method, calls, wantCalls)
						}
					}
				}
			})
		}
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

func TestPreflightRequestHeaderFields(t *testing.T) {
	cases := []struct {
		name    string
		values  []string
		want    []string
		present bool
		denied  bool
	}{
		{name: "leading_empty", values: []string{"", "authorization", "content-type"}, want: []string{"", "authorization", "content-type"}, present: true},
		{name: "leading_empties", values: []string{"", "", "authorization"}, want: []string{"", "", "authorization"}, present: true},
		{name: "nonempty", values: []string{"authorization"}, want: []string{"authorization"}, present: true},
		{name: "empty", values: []string{""}, present: true},
		{name: "all_empty", values: []string{"", ""}, present: true},
		{name: "absent"},
		{name: "nil", present: true},
		{name: "empty_slice", values: []string{}, present: true},
		{name: "denied_later", values: []string{"", "x-denied"}, want: []string{"", "x-denied"}, present: true, denied: true},
	}
	for _, wildcard := range []bool{false, true} {
		name := "explicit"
		allowedHeaders := []string{"Authorization", "Content-Type"}
		if wildcard {
			name = "wildcard"
			allowedHeaders = []string{"*"}
		}
		t.Run(name, func(t *testing.T) {
			s := New(Options{AllowedHeaders: allowedHeaders})
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodOptions, "http://example.com/", nil)
					req.Header.Set("Origin", "http://foobar.com")
					req.Header.Set("Access-Control-Request-Method", http.MethodGet)
					if tc.present {
						req.Header["Access-Control-Request-Headers"] = tc.values
					}
					res := httptest.NewRecorder()
					s.Handler(testHandler).ServeHTTP(res, req)
					assertResponse(t, res, http.StatusNoContent)

					want := http.Header{
						"Vary": {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
					}
					if !tc.denied || wildcard {
						want["Access-Control-Allow-Origin"] = []string{"*"}
						want["Access-Control-Allow-Methods"] = []string{http.MethodGet}
						if tc.want != nil {
							want["Access-Control-Allow-Headers"] = tc.want
						}
					}
					assertHeaders(t, res.Header(), want)
				})
			}
		})
	}
}

func TestPreflightEmptyLeadingHeaderOverHTTP(t *testing.T) {
	for _, allowedHeaders := range [][]string{{"Authorization"}, {"*"}} {
		t.Run(allowedHeaders[0], func(t *testing.T) {
			s := httptest.NewServer(New(Options{AllowedHeaders: allowedHeaders}).Handler(testHandler))
			defer s.Close()
			req, err := http.NewRequest(http.MethodOptions, s.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Origin", "http://foobar.com")
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
			req.Header["Access-Control-Request-Headers"] = []string{"", "authorization"}
			res, err := s.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusNoContent {
				t.Errorf("Status = %d, want %d", res.StatusCode, http.StatusNoContent)
			}
			want := []string{"", "authorization"}
			if got := res.Header.Values("Access-Control-Allow-Headers"); !slices.Equal(got, want) {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, want)
			}
		})
	}
}

type testLogger struct {
	buf *bytes.Buffer
}

func (l *testLogger) Printf(format string, v ...any) {
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
	if s.allowedHeaders.Size() == 0 {
		t.Error("c.allowedHeaders should be empty when Default")
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
	req.Header.Add("Origin", "http://example.com")

	s.handlePreflight(res, req)

	assertHeaders(t, res.Header(), http.Header{
		"Vary": {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
	})
}

func TestHandleActualRequestInvalidOriginAbortion(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foo.com"},
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com")

	s.handleActualRequest(res, req)

	assertHeaders(t, res.Header(), http.Header{
		"Vary": {"Origin"},
	})
}

func TestHandleActualRequestInvalidMethodAbortion(t *testing.T) {
	s := New(Options{
		AllowedMethods:   []string{"POST"},
		AllowCredentials: true,
	})
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://example.com")

	s.handleActualRequest(res, req)

	assertHeaders(t, res.Header(), http.Header{
		"Vary": {"Origin"},
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
