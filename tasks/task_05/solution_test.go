package main

import "testing"

func TestHardCapacity_InsertIntoEmpty(t *testing.T) {
	c := NewCache[string, int](1)

	ok := c.Set("a", 0)

	if !ok {
		t.Fatal("Set into empty cache must succeed")
	}
}

func TestHardCapacity_RejectsNewKeyWhenFull(t *testing.T) {
	c := NewCache[string, int](1)
	c.Set("a", 0)

	ok := c.Set("b", 2)

	if ok {
		t.Error("full cache must reject new key")
	}
	if v, found := c.Get("b"); v != 0 || found {
		t.Errorf("Get(b) = %d, %t; want 0, false", v, found)
	}
	if v, found := c.Get("a"); v != 0 || !found {
		t.Errorf("Get(a) = %d, %t; want 0, true: rejection must not evict", v, found)
	}
}

func TestCapacityTwo(t *testing.T) {
	c := NewCache[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)

	rejected := !c.Set("c", 3)
	updated := c.Set("b", 20)

	if !rejected {
		t.Error("third key must be rejected")
	}
	if !updated {
		t.Error("update at capacity must succeed")
	}
	for _, tc := range []struct {
		key   string
		want  int
		found bool
	}{
		{key: "a", want: 1, found: true},
		{key: "b", want: 20, found: true},
		{key: "c", want: 0, found: false},
	} {
		if v, found := c.Get(tc.key); v != tc.want || found != tc.found {
			t.Errorf("Get(%s) = %d, %t; want %d, %t", tc.key, v, found, tc.want, tc.found)
		}
	}
}

func TestHardCapacity_UpdatesExistingKeyWhenFull(t *testing.T) {
	c := NewCache[string, int](1)
	c.Set("a", 0)

	ok := c.Set("a", 3)

	if !ok {
		t.Error("update at capacity must succeed")
	}
	if v, found := c.Get("a"); v != 3 || !found {
		t.Errorf("Get(a) = %d, %t; want 3, true", v, found)
	}
}

func TestDisabled(t *testing.T) {
	for _, tc := range []struct {
		name     string
		capacity int
	}{
		{name: "negative capacity", capacity: -1},
		{name: "zero capacity", capacity: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCache[int, string](tc.capacity)

			ok := c.Set(1, "x")

			if ok {
				t.Error("disabled cache must reject Set")
			}
			if v, found := c.Get(1); v != "" || found {
				t.Errorf("Get(1) = %q, %t; want \"\", false", v, found)
			}
		})
	}
}

func TestStructKeysAndNilValues(t *testing.T) {
	type key struct{ N int }
	c := NewCache[key, []int](2)

	ok := c.Set(key{1}, nil)

	if !ok {
		t.Fatal("Set with nil value must succeed")
	}
	if v, found := c.Get(key{1}); v != nil || !found {
		t.Errorf("Get(key{1}) = %v, %t; want nil, true", v, found)
	}
	if v, found := c.Get(key{2}); v != nil || found {
		t.Errorf("Get(key{2}) = %v, %t; want nil, false", v, found)
	}
}
