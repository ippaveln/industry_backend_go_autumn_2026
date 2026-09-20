package main

import (
	"math/rand"
	"testing"
)

var _ LRU[string, int] = (*LRUCache[string, int])(nil)

func TestWriteDoesNotRefresh(t *testing.T) {
	c := NewLRUCache[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("a", 3)
	c.Set("c", 4)
	if _, ok := c.Get("a"); ok {
		t.Error("overwrite must not refresh a")
	}
	if v, ok := c.Get("b"); v != 2 || !ok {
		t.Error("b was incorrectly evicted")
	}
}
func TestReadRefreshAndDisabled(t *testing.T) {
	c := NewLRUCache[int, int](2)
	c.Set(1, 0)
	c.Set(2, 2)
	if v, ok := c.Get(1); v != 0 || !ok {
		t.Fatal("stored zero")
	}
	c.Get(9)
	c.Set(3, 3)
	if _, ok := c.Get(2); ok {
		t.Fatal("read recency")
	}
	for _, n := range []int{-1, 0} {
		d := NewLRUCache[int, int](n)
		d.Set(1, 1)
		if v, ok := d.Get(1); v != 0 || ok {
			t.Fatal("disabled")
		}
	}
	d := NewLRUCache[int, int](1)
	d.Set(1, 1)
	d.Set(1, 2)
	if v, ok := d.Get(1); v != 2 || !ok {
		t.Fatal("overwrite")
	}
	d.Set(2, 2)
	if _, ok := d.Get(1); ok {
		t.Fatal("capacity one")
	}
}
func TestLRUModel(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	for cap := 1; cap <= 5; cap++ {
		c := NewLRUCache[int, int](cap)
		m := map[int]int{}
		order := []int{}
		for step := 0; step < 500; step++ {
			k := r.Intn(10)
			if r.Intn(2) == 0 {
				v := r.Intn(100)
				c.Set(k, v)
				if _, ok := m[k]; !ok {
					if len(order) == cap {
						delete(m, order[0])
						order = order[1:]
					}
					order = append(order, k)
				}
				m[k] = v
			} else {
				v, ok := c.Get(k)
				w, exists := m[k]
				if v != w || ok != exists {
					t.Fatalf("cap=%d step=%d key=%d", cap, step, k)
				}
				if exists {
					for j, x := range order {
						if x == k {
							order = append(order[:j], order[j+1:]...)
							break
						}
					}
					order = append(order, k)
				}
			}
		}
	}
}
