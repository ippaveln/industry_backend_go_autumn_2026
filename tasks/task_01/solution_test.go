package main

import "testing"

func TestGreeting(t *testing.T) {
	for _, c := range []struct{ in, want string }{{"", "Hello, World!"}, {"Alice", "Hello, Alice!"}, {" \tАнна\u2003", "Hello, Анна!"}, {"\u00a0\n", "Hello, World!"}, {" A  B ", "Hello, A  B!"}} {
		if got := greet(c.in); got != c.want {
			t.Errorf("%q: got %q want %q", c.in, got, c.want)
		}
	}
}
