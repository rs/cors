package cors

import (
	"net/http"
	"testing"
)

type FakeResponse struct {
	header http.Header
}

func (r FakeResponse) Header() http.Header {
	return r.header
}

func (r FakeResponse) WriteHeader(n int) {
}

func (r FakeResponse) Write(b []byte) (n int, err error) {
	return len(b), nil
}

const (
	headerOrigin  = "Origin"
	headerACRM    = "Access-Control-Request-Method"
	headerACRH    = "Access-Control-Request-Headers"
	dummyEndpoint = "http://example.com/foo"
	dummyOrigin   = "https://somedomain.com"
)

func BenchmarkWithout(b *testing.B) {
	res := FakeResponse{http.Header{}}
	req, _ := http.NewRequest(http.MethodGet, dummyEndpoint, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(res.header)
		testHandler.ServeHTTP(res, req)
	}
}

func BenchmarkDefault(b *testing.B) {
	res := FakeResponse{http.Header{}}
	req, _ := http.NewRequest(http.MethodGet, dummyEndpoint, nil)
	req.Header.Add(headerOrigin, dummyOrigin)
	handler := Default().Handler(testHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(res.header)
		handler.ServeHTTP(res, req)
	}
}

func BenchmarkAllowedOrigin(b *testing.B) {
	res := FakeResponse{http.Header{}}
	req, _ := http.NewRequest(http.MethodGet, dummyEndpoint, nil)
	req.Header.Add(headerOrigin, dummyOrigin)
	c := New(Options{
		AllowedOrigins: []string{dummyOrigin},
	})
	handler := c.Handler(testHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(res.header)
		handler.ServeHTTP(res, req)
	}
}

func BenchmarkPreflight(b *testing.B) {
	res := FakeResponse{http.Header{}}
	req, _ := http.NewRequest(http.MethodOptions, dummyEndpoint, nil)
	req.Header.Add(headerOrigin, dummyOrigin)
	req.Header.Add(headerACRM, http.MethodGet)
	handler := Default().Handler(testHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(res.header)
		handler.ServeHTTP(res, req)
	}
}

func BenchmarkPreflightHeader(b *testing.B) {
	res := FakeResponse{http.Header{}}
	req, _ := http.NewRequest(http.MethodOptions, dummyEndpoint, nil)
	req.Header.Add(headerOrigin, dummyOrigin)
	req.Header.Add(headerACRM, http.MethodGet)
	req.Header.Add(headerACRH, "Accept")
	handler := Default().Handler(testHandler)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(res.header)
		handler.ServeHTTP(res, req)
	}
}

func clear(h http.Header) {
	for k := range h {
		delete(h, k)
	}
}
