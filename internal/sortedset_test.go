package internal

import (
	"testing"
)

func TestSortedSet(t *testing.T) {
	cases := []struct {
		desc       string
		elems      []string
		combined   string
		subsets    []string
		notSubsets []string
		wantSize   int
	}{
		{
			desc:     "empty set",
			combined: "",
			notSubsets: []string{
				"bar",
				"bar,foo",
			},
			wantSize: 0,
		}, {
			desc:     "singleton set",
			elems:    []string{"foo"},
			combined: "foo",
			subsets: []string{
				"",
				"foo",
			},
			notSubsets: []string{
				"bar",
				"bar,foo",
			},
			wantSize: 1,
		}, {
			desc:     "no dupes",
			elems:    []string{"foo", "bar", "baz"},
			combined: "bar,baz,foo",
			subsets: []string{
				"",
				"bar",
				"baz",
				"foo",
				"bar,baz",
				"bar,foo",
				"baz,foo",
				"bar,baz,foo",
			},
			notSubsets: []string{
				"qux",
				"bar,baz,baz",
				"qux,baz",
				"qux,foo",
				"quxbaz,foo",
			},
			wantSize: 3,
		}, {
			desc:     "some dupes",
			elems:    []string{"foo", "bar", "bar", "foo", "e"},
			combined: "bar,e,foo",
			subsets: []string{
				"",
				"bar",
				"e",
				"foo",
				"bar,foo",
				"bar,e",
				"e,foo",
				"bar,e,foo",
			},
			notSubsets: []string{
				"qux",
				"qux,bar",
				"qux,foo",
				"qux,baz,foo",
			},
			wantSize: 3,
		},
	}
	for _, tc := range cases {
		f := func(t *testing.T) {
			elems := clone(tc.elems)
			s := NewSortedSet(tc.elems...)
			size := s.Size()
			if s.Size() != tc.wantSize {
				const tmpl = "NewSortedSet(%#v...).Size(): got %d; want %d"
				t.Errorf(tmpl, elems, size, tc.wantSize)
			}
			combined := s.String()
			if combined != tc.combined {
				const tmpl = "NewSortedSet(%#v...).String(): got %q; want %q"
				t.Errorf(tmpl, elems, combined, tc.combined)
			}
			for _, sub := range tc.subsets {
				if !s.Subsumes(sub) {
					const tmpl = "%q is not a subset of %q, but should be"
					t.Errorf(tmpl, sub, s)
				}
			}
			for _, notSub := range tc.notSubsets {
				if s.Subsumes(notSub) {
					const tmpl = "%q is a subset of %q, but should not be"
					t.Errorf(tmpl, notSub, s)
				}
			}
		}
		t.Run(tc.desc, f)
	}
}

// adapted from https://pkg.go.dev/slices#Clone
// TODO: when updating go directive to 1.21 or later,
// use slices.Clone instead.
func clone(s []string) []string {
	// The s[:0:0] preserves nil in case it matters.
	return append(s[:0:0], s...)
}
