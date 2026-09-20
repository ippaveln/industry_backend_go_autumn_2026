package main

import (
	"errors"
	"strconv"
	"testing"
)

func TestSignedFizzBuzz(t *testing.T) {
	if s, e := fizzBuzz(0); s != "" || !errors.Is(e, ErrZero) {
		t.Fatalf("zero: %q %v", s, e)
	}
	for n := -100; n <= 100; n++ {
		if n == 0 {
			continue
		}
		w := strconv.Itoa(n)
		if n%15 == 0 {
			w = "FizzBuzz"
		} else if n%3 == 0 {
			w = "Fizz"
		} else if n%5 == 0 {
			w = "Buzz"
		}
		if s, e := fizzBuzz(n); s != w || e != nil {
			t.Errorf("n=%d: %q %v, want %q", n, s, e, w)
		}
	}
	n := -int(^uint(0)>>1) - 1
	if s, e := fizzBuzz(n); s != strconv.Itoa(n) || e != nil {
		t.Fatalf("min int %q %v", s, e)
	}
}
