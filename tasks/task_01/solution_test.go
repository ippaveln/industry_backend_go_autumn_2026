package main

import "testing"

func TestGreeting(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "empty name", in: "", want: "Hello, World!"},
		{name: "plain name", in: "Alice", want: "Hello, Alice!"},
		{name: "unicode spaces trimmed", in: " \tАнна ", want: "Hello, Анна!"},
		{name: "only whitespace", in: " \n", want: "Hello, World!"},
		{name: "inner spaces kept", in: " A  B ", want: "Hello, A  B!"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := greet(tc.in)

			if got != tc.want {
				t.Errorf("greet(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
