package cors

import (
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	s := convert([]string{"A", "b", "C"}, strings.ToLower)
	e := []string{"a", "b", "c"}
	if s[0] != e[0] || s[1] != e[1] || s[2] != e[2] {
		t.Errorf("%v != %v", s, e)
	}
}

func TestToHeader(t *testing.T) {
	h := toHeader("mY-header")
	e := "My-Header"
	if h != e {
		t.Errorf("%v != %v", h, e)
	}
}
