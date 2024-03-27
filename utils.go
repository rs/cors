package cors

import (
	"net/http"
	"strings"
)

type converter func(string) string

type wildcard struct {
	prefix string
	suffix string
}

func (w wildcard) match(s string) bool {
	return len(s) >= len(w.prefix)+len(w.suffix) && strings.HasPrefix(s, w.prefix) && strings.HasSuffix(s, w.suffix)
}

// convert converts a list of string using the passed converter function
func convert(s []string, c converter) []string {
	out, _ := convertDidCopy(s, c)
	return out
}

// convertDidCopy is same as convert but returns true if it copied the slice
func convertDidCopy(s []string, c converter) ([]string, bool) {
	out := s
	copied := false
	for i, v := range s {
		if !copied {
			v2 := c(v)
			if v2 != v {
				out = make([]string, len(s))
				copy(out, s[:i])
				out[i] = v2
				copied = true
			}
		} else {
			out[i] = c(v)
		}
	}
	return out, copied
}

func first(hdrs http.Header, k string) ([]string, bool) {
	v, found := hdrs[k]
	if !found || len(v) == 0 {
		return nil, false
	}
	return v[:1], true
}
