package main

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestDeltaStats(t *testing.T) {
	for _, c := range []struct {
		in   []int64
		want Stats
	}{{nil, Stats{}}, {[]int64{9}, Stats{}}, {[]int64{10, 13, 8, 8}, Stats{3, -2, -5, 3}}, {[]int64{4, 9, 11}, Stats{2, 7, 2, 5}}, {[]int64{9, 4, 1}, Stats{2, -8, -5, -3}}, {[]int64{1, 1}, Stats{1, 0, 0, 0}}} {
		before := append([]int64(nil), c.in...)
		if got := Calc(c.in); got != c.want {
			t.Errorf("%v: %+v != %+v", c.in, got, c.want)
		}
		if !reflect.DeepEqual(before, c.in) {
			t.Fatal("input mutated")
		}
	}
}
func TestDeltaModel(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	for n := 2; n < 100; n++ {
		a := make([]int64, n)
		for i := range a {
			a[i] = r.Int63n(1000) - 500
		}
		d := make([]int64, n-1)
		for i := range d {
			d[i] = a[i+1] - a[i]
		}
		w := Stats{Count: len(d), Min: d[0], Max: d[0]}
		for _, v := range d {
			w.Sum += v
			w.Min = min(w.Min, v)
			w.Max = max(w.Max, v)
		}
		if got := Calc(a); got != w {
			t.Fatal(got, w)
		}
	}
}
