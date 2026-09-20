package main

import "testing"

func TestHardCapacity(t *testing.T) {
	c := NewCache[string, int](1)
	if !c.Set("a", 0) {
		t.Fatal("insert")
	}
	if c.Set("b", 2) {
		t.Error("full cache must reject new key")
	}
	if v, ok := c.Get("b"); v != 0 || ok {
		t.Error("rejection changed cache")
	}
	if !c.Set("a", 3) {
		t.Error("update at capacity")
	}
	if v, ok := c.Get("a"); v != 3 || !ok {
		t.Error(v, ok)
	}
}
func TestDisabledAndTypes(t *testing.T) {
	for _, n := range []int{-1, 0} {
		c := NewCache[int, string](n)
		if c.Set(1, "x") {
			t.Fatal("disabled")
		}
		if v, ok := c.Get(1); v != "" || ok {
			t.Fatal(v, ok)
		}
	}
	type key struct{ N int }
	c := NewCache[key, []int](2)
	if !c.Set(key{1}, nil) {
		t.Fatal("insert nil")
	}
	if v, ok := c.Get(key{1}); v != nil || !ok {
		t.Fatal(v, ok)
	}
	if v, ok := c.Get(key{2}); v != nil || ok {
		t.Fatal(v, ok)
	}
}
