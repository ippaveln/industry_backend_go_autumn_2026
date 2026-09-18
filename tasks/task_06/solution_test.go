package main

import (
	"strconv"
	"strings"
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
		t.Errorf("Get(b) = %d, %t; want 2, true", v, ok)
	}
}

func TestStoresZeroValue(t *testing.T) {
	c := NewLRUCache[int, int](2)
	c.Set(1, 0)

	v, ok := c.Get(1)

	if v != 0 || !ok {
		t.Fatalf("Get(1) = %d, %t; want 0, true", v, ok)
	}
}

func TestStoresNilPointer(t *testing.T) {
	c := NewLRUCache[string, *int](1)
	c.Set("p", nil)

	v, ok := c.Get("p")

	if v != nil || !ok {
		t.Fatalf("Get(p) = %v, %t; want nil, true", v, ok)
	}
}

func TestReadRefreshesRecency(t *testing.T) {
	c := NewLRUCache[int, int](2)
	c.Set(1, 0)
	c.Set(2, 2)
	c.Get(1)
	c.Get(9)

	c.Set(3, 3)

	if _, ok := c.Get(2); ok {
		t.Fatal("least recently read key 2 must be evicted")
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
			c := NewLRUCache[int, int](tc.capacity)
			for k := range 100 {
				c.Set(k%3, k)
			}

			for k := range 3 {
				if v, ok := c.Get(k); v != 0 || ok {
					t.Fatalf("Get(%d) = %d, %t; want 0, false", k, v, ok)
				}
			}
		})
	}
}

func TestCapacityOne_Overwrite(t *testing.T) {
	c := NewLRUCache[int, int](1)
	c.Set(1, 1)

	c.Set(1, 2)

	if v, ok := c.Get(1); v != 2 || !ok {
		t.Fatalf("Get(1) = %d, %t; want 2, true", v, ok)
	}
}

func TestCapacityOne_Evicts(t *testing.T) {
	c := NewLRUCache[int, int](1)
	c.Set(1, 1)

	c.Set(2, 2)

	if _, ok := c.Get(1); ok {
		t.Fatal("key 1 must be evicted")
	}
}

// Each trace runs on a fresh cache: "k=v" is Set(k, v), "k?v" is Get(k)
// that must return v, true, and "k?" is Get(k) that must miss.
func TestScenarios(t *testing.T) {
	for _, tc := range []struct {
		name     string
		capacity int
		trace    string
	}{
		{"readme example", 2, "0=1 1=2 0=3 2=4 0? 1?2 2?4"},
		{"readme example with read", 2, "0=1 1=2 0=3 0?3 2=4 1? 0?3 2?4"},
		{"evicts in insertion order", 3, "1=1 2=2 3=3 4=4 1? 2?2 3?3 4?4"},
		{"reads reorder eviction", 3, "1=1 2=2 3=3 1?1 2?2 4=4 3? 5=5 1? 2?2 4?4 5?5"},
		{"overwrites keep order", 3, "1=1 2=2 3=3 1=10 1=11 1=12 4=4 1? 2?2 3?3 4?4"},
		{"miss keeps order", 2, "1=1 2=2 7? 8? 9? 3=3 1? 2?2 3?3"},
		{"read of evicted key misses", 2, "1=1 2=2 3=3 1? 1? 2?2 3?3"},
		{"reinserted key is fresh", 2, "1=1 2=2 3=3 1=5 2? 3?3 1?5"},
		{"overwrite after read keeps read order", 2, "1=1 2=2 1?1 2=3 3=3 1?1 2? 3?3"},
		{"repeated reads", 3, "1=1 2=2 3=3 1?1 1?1 1?1 4=4 2? 3?3 1?1 4?4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runTrace(t, tc.capacity, tc.trace)
		})
	}
}

func TestTraces(t *testing.T) {
	for i, tc := range []struct {
		capacity int
		trace    string
	}{
		{1, "0? 0=6 1=46 1=53 0? 0? 1?53 0=90 1? 0?90 0=91 0=7 1=73 0=28 1? 0?28 0=31 0=72 1? 0=0 1? 1? 0=34 1? 1? 0=11 1=40 0? 1?40 1?40 1=3 0=22 1? 0=24 0=59 1=33"},
		{1, "0=34 0?34 1=97 2? 1?97 1?97 2? 2? 1?97 3=13 1? 3=19 0? 2=91 0? 2?91 1=73 0=8 0=11 1=61 0=67 1=36 3=95 0? 2? 3?95 0=27 3=78 0? 1=52 0? 2=52 1=85 0? 0? 2=3"},
		{1, "3? 0=37 0=38 3=26 5=33 2? 1? 1? 2? 5=36 2? 3? 1? 2=29 4? 5=58 0? 1? 0? 4=64 3? 2=21 0=46 2=16 2=88 3=39 4? 3?39 3?39 2? 0? 3=90 2=47 0? 0? 3?"},
		{1, "2? 6? 0? 2? 5? 0? 3=39 2? 5? 5? 1? 5? 1? 3=39 7? 1=23 2=48 1=9 2? 0=85 3=1 2? 3=28 7? 5? 1? 2=86 0=67 6? 4=97 6=46 6?46 6?46 6?46 7? 1=4"},
		{2, "0=41 2=30 0?41 2=53 1? 2?53 2=62 2=25 1? 2?25 2?25 2?25 1=47 1=17 1=99 0=23 0?23 0?23 1?99 1=4 0?23 1=7 0?23 1?7 1=12 2=19 1=43 2=14 2?14 1=99 1=22 0? 0? 1=16 1=13 0?"},
		{2, "3? 2=54 0=71 4? 0=20 4=6 0?20 2=59 0?20 0?20 3=29 1? 1? 2=58 0=49 2?58 1? 4? 0=61 4? 0?61 4? 2?58 4? 2=52 3? 4? 3=41 0? 3?41 4? 1? 4=48 1=81 1?81 4?48"},
		{2, "3? 6=9 6?9 2=26 3? 3? 4? 1? 3=11 0? 0=24 2=5 2=70 0?24 3=99 1? 1? 4? 6? 1? 2? 5? 6? 3=25 5? 3?25 2=35 1? 5? 5? 4? 0? 3=45 5? 2=93 5=53"},
		{2, "4? 8? 0? 6=50 2=0 3? 0? 3? 6=72 1=13 8? 1=2 8? 3? 5=20 8? 6? 8? 7=99 4? 4=80 7?99 6? 0? 6? 0=91 7=60 6=8 5=99 3? 0? 6=34 7? 0=1 7=41 6?"},
		{3, "0? 1? 3? 1? 1=35 1=94 0=71 0?71 2? 0?71 3? 2? 0=13 3? 1=13 3? 1=0 2=92 3=28 0=53 3?28 3?28 2=3 0?53 1? 0?53 0?53 1=9 1?9 2? 3?28 0?53 1?9 1=62 2=12 1?62"},
		{3, "4? 3=42 5=36 5=1 1? 0? 1=57 1=77 2? 3?42 0=45 5=40 4=59 3? 5=30 5=56 5=15 1? 0=80 0?80 0=13 1? 2? 3? 0?13 0?13 0=76 5?15 2? 4=69 4=29 4?29 1? 5=81 1? 4=78"},
		{3, "7=23 1? 2=26 4? 2?26 5=63 6=34 4? 7=82 1=41 6?34 2=42 1=47 5? 6?34 5? 2=31 2?31 2?31 3? 5=72 7=63 3=6 4=62 7=70 3=15 1? 6=49 6?49 3?15 1=10 6?49 2? 6?49 3?15 4=82"},
		{3, "5? 0? 6=84 0? 9=44 4? 9=84 5=71 3=97 1? 0? 4? 1? 7=22 8? 5=22 6? 7=27 8=31 2? 4? 9=25 1? 5? 9=64 0? 5? 7=36 4? 5=4 6? 4=99 5?4 0=31 9? 8?"},
		{4, "3? 0=45 0?45 0=96 2? 1? 0?96 4? 4? 2? 4=16 3? 0=92 2? 3? 1? 4=76 2? 2? 3=26 1? 0?92 0?92 0?92 1? 0=96 0?96 2? 2=4 4?76 1=22 1?22 1?22 1?22 3? 0?96"},
		{4, "4? 1=43 5=73 1?43 1?43 3=17 0? 2=77 5?73 6? 5=53 2=0 2?0 0? 4? 2?0 2?0 0? 5?53 1?43 0=29 6? 1=89 5=41 2?0 2?0 6? 1=97 3=58 2?0 6=0 6?0 2=15 2?15 0?29 0?29"},
		{4, "2=78 6? 5? 2=83 3? 4=41 5? 0? 6? 3=41 0=94 5=80 6? 2=84 6=57 2?84 8? 3=65 7=33 6=4 7?33 4? 2=62 3=24 6?4 3=1 4? 2?62 0=54 1=24 2?62 2?62 0=19 3=8 0=20 1=97"},
		{4, "0? 9? 1? 9=63 8=47 7=29 6=77 3? 6?77 6=81 3? 8=92 8?92 1? 10? 5=29 9? 1? 2=16 5?29 3=26 4=41 4?41 0? 3?26 2?16 9? 9=53 6=50 7=49 5? 3? 1=43 6?50 7=47 6?50"},
		{5, "3? 3? 5=4 5=75 4=99 0=68 2=67 1=58 5?75 1=52 2?67 2=75 4?99 2=88 3=30 4?99 2=8 1=17 5?75 5=21 5?21 3?30 4=95 4?95 4=35 5?21 4?35 2?8 2=7 5?21 3?30 0? 3?30 0? 1=21 3?30"},
		{5, "0? 5? 2=5 0? 3? 3=69 7? 3=84 7? 6=47 1=43 0=34 1?43 3?84 1?43 4=27 2=44 5=87 7=46 1=48 5?87 7=77 5?87 0=53 4=66 1? 4=77 0=4 0?4 2?44 5?87 3=90 1? 4=51 0=81 2?44"},
		{5, "5? 0=44 6? 4=23 2? 4=87 7=45 9? 2? 9? 1? 5=94 4=92 9? 9? 0?44 0?44 5=68 0?44 3? 2? 1=34 3=18 5?68 6? 6=85 8=68 5?68 9=4 8?68 4=33 2=14 0=13 0?13 7=10 0=59"},
		{5, "1=8 8? 4=11 10=4 8? 3? 0? 10?4 7? 0? 2? 3? 11=22 4=90 10?4 10?4 5? 9? 11=68 5? 4=71 2? 6? 4?71 10?4 10?4 4?71 11=69 3? 0? 6? 2=4 8? 5? 9? 9?"},
		{6, "3? 0=29 4=66 4?66 0?29 2? 3? 6? 5? 6? 2? 5=62 2=27 4=34 6? 6=13 1? 2?27 0?29 2?27 6?13 4?34 3? 4=66 1? 5=85 0?29 0?29 5?85 2?27 3? 1? 2?27 4?66 5?85 1=7"},
		{6, "7=58 3? 4=27 2? 5=68 2? 1=42 8=24 1=1 7?58 8?24 7?58 7?58 4?27 8?24 1=88 8?24 5=31 1?88 2? 2? 7?58 6? 8?24 8=73 7?58 5?31 1?88 3? 6=66 5=12 7?58 6?66 8?73 7=88 1?88"},
		{6, "10=90 5? 7=1 1? 3=93 7=93 8? 1=11 2? 4=14 5? 10?90 4?14 8? 4?14 2? 0? 7=60 9=7 7?60 7?60 9?7 3=88 9=32 9?32 1=18 7?60 1?18 9?32 2? 1=40 5? 9?32 1=15 5=46 2=3"},
		{6, "8? 2=99 11? 3? 4? 7=55 12? 0? 8? 7=80 5? 5? 4? 11? 10? 9=83 12? 8=24 10=11 0? 2?99 2?99 5=58 4? 7?80 5?58 7=89 9?83 3? 6? 0? 7?89 0=32 6? 6? 6=17"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			runTrace(t, tc.capacity, tc.trace)
		})
	}
}

func TestLargeCapacity(t *testing.T) {
	const capacity = 1000
	c := NewLRUCache[int, int](capacity)
	for k := range 2 * capacity {
		c.Set(k, -k)
	}
	for k := range capacity {
		if _, ok := c.Get(k); ok {
			t.Fatalf("Get(%d) hit, key must be evicted", k)
		}
	}
	for k := 2*capacity - 1; k >= capacity; k-- {
		if v, ok := c.Get(k); v != -k || !ok {
			t.Fatalf("Get(%d) = %d, %t; want %d, true", k, v, ok, -k)
		}
	}

	// Reads went from the newest key down, so the newest keys are now the oldest.
	for k := 2 * capacity; k < 2*capacity+capacity/2; k++ {
		c.Set(k, -k)
	}
	for k := capacity; k < 2*capacity+capacity/2; k++ {
		_, ok := c.Get(k)
		if want := k < capacity+capacity/2 || k >= 2*capacity; ok != want {
			t.Fatalf("Get(%d) hit = %t, want %t", k, ok, want)
		}
	}
}

func runTrace(t *testing.T, capacity int, trace string) {
	t.Helper()
	c := NewLRUCache[int, int](capacity)
	for step, op := range strings.Fields(trace) {
		if k, v, ok := strings.Cut(op, "="); ok {
			c.Set(atoi(t, k), atoi(t, v))
			continue
		}
		k, v, _ := strings.Cut(op, "?")
		got, ok := c.Get(atoi(t, k))
		if v == "" {
			if ok {
				t.Fatalf("capacity=%d step=%d: Get(%s) = %d, true; want miss\ntrace: %s", capacity, step, k, got, trace)
			}
			continue
		}
		if want := atoi(t, v); got != want || !ok {
			t.Fatalf("capacity=%d step=%d: Get(%s) = %d, %t; want %d, true\ntrace: %s", capacity, step, k, got, ok, want, trace)
		}
	}
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("bad trace number %q", s)
	}
	return n
}
