package main

import "testing"

func TestGreeting(t *testing.T) {
	want := "Hello, World!"

	got := greet()

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
