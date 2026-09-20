package main

import "testing"

func TestGreeting(t *testing.T) {
	if greet() != "Hello, World!" {
		t.Fatal("exact greeting required")
	}
}
