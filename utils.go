package cors

import (
	"strings"
)

type converter func(string) string

// convert converts a list of string using the passed converter function
func convert(s []string, c converter) []string {
	out := []string{}
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}

// toHeader converts an arbitrary formatted string to a HTTP header formatted string
// i.e.: my-header becomes My-Header
func toHeader(header string) string {
	chunks := strings.Split(header, "")
	upNext := true
	for pos, char := range chunks {
		if upNext {
			chunks[pos] = strings.ToUpper(char)
			upNext = false
		} else if char == "-" {
			upNext = true
		} else {
			chunks[pos] = strings.ToLower(char)
		}
	}
	return strings.Join(chunks, "")
}
