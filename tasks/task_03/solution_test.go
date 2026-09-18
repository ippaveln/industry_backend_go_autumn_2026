package main

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestFizzBuzz_Zero(t *testing.T) {
	got, err := fizzBuzz(0)

	if got != "" || !errors.Is(err, ErrZero) {
		t.Fatalf("fizzBuzz(0) = %q, %v; want \"\", %v", got, err, ErrZero)
	}
}

func TestFizzBuzz_Table(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want string
	}{
		{-30, "FizzBuzz"}, {-29, "-29"}, {-28, "-28"}, {-27, "Fizz"}, {-26, "-26"},
		{-25, "Buzz"}, {-24, "Fizz"}, {-23, "-23"}, {-22, "-22"}, {-21, "Fizz"},
		{-20, "Buzz"}, {-19, "-19"}, {-18, "Fizz"}, {-17, "-17"}, {-16, "-16"},
		{-15, "FizzBuzz"}, {-14, "-14"}, {-13, "-13"}, {-12, "Fizz"}, {-11, "-11"},
		{-10, "Buzz"}, {-9, "Fizz"}, {-8, "-8"}, {-7, "-7"}, {-6, "Fizz"},
		{-5, "Buzz"}, {-4, "-4"}, {-3, "Fizz"}, {-2, "-2"}, {-1, "-1"},
		{1, "1"}, {2, "2"}, {3, "Fizz"}, {4, "4"}, {5, "Buzz"},
		{6, "Fizz"}, {7, "7"}, {8, "8"}, {9, "Fizz"}, {10, "Buzz"},
		{11, "11"}, {12, "Fizz"}, {13, "13"}, {14, "14"}, {15, "FizzBuzz"},
		{16, "16"}, {17, "17"}, {18, "Fizz"}, {19, "19"}, {20, "Buzz"},
		{21, "Fizz"}, {22, "22"}, {23, "23"}, {24, "Fizz"}, {25, "Buzz"},
		{26, "26"}, {27, "Fizz"}, {28, "28"}, {29, "29"}, {30, "FizzBuzz"},
		{math.MaxInt, "9223372036854775807"},
		{math.MaxInt - 1, "Fizz"},
		{9223372036854775805, "Buzz"},
		{9223372036854775800, "FizzBuzz"},
		{math.MinInt, "-9223372036854775808"},
		{math.MinInt + 1, "-9223372036854775807"},
		{-9223372036854775806, "Fizz"},
		{-9223372036854775805, "Buzz"},
		{-9223372036854775801, "-9223372036854775801"},
		{2147483647, "2147483647"},
		{-2147483648, "-2147483648"},
		{1099511627776, "1099511627776"},
		{-1099511627776, "-1099511627776"},
		{999999999999990, "FizzBuzz"},
		{-999999999999990, "FizzBuzz"},
		{1000000005, "FizzBuzz"},
		{-1000000003, "-1000000003"},
		{123456789012, "Fizz"},
		{-3000000000000000001, "-3000000000000000001"},
	} {
		got, err := fizzBuzz(tc.n)

		if got != tc.want || err != nil {
			t.Errorf("fizzBuzz(%d) = %q, %v; want %q, nil", tc.n, got, err, tc.want)
		}
	}
}

func TestFizzBuzz_MinInt(t *testing.T) {
	n := math.MinInt
	want := strconv.Itoa(n)

	got, err := fizzBuzz(n)

	if got != want || err != nil {
		t.Fatalf("fizzBuzz(%d) = %q, %v; want %q, nil", n, got, err, want)
	}
}

// Results repeat with period 15, and a number keeps its own spelling.
func TestFizzBuzz_Period(t *testing.T) {
	words := map[string]bool{"Fizz": true, "Buzz": true, "FizzBuzz": true}

	for n := -3000; n <= 3000; n++ {
		if n == 0 || n+15 == 0 {
			continue
		}
		got, err := fizzBuzz(n)
		if err != nil {
			t.Fatalf("fizzBuzz(%d) returned error %v", n, err)
		}
		if !words[got] && got != strconv.Itoa(n) {
			t.Fatalf("fizzBuzz(%d) = %q, want a word or %q", n, got, strconv.Itoa(n))
		}
		next, _ := fizzBuzz(n + 15)
		if words[got] != words[next] || words[got] && got != next {
			t.Fatalf("fizzBuzz(%d) = %q, but fizzBuzz(%d) = %q", n, got, n+15, next)
		}
	}
}

func TestFizzBuzz_Symmetric(t *testing.T) {
	for n := 1; n <= 3000; n++ {
		pos, _ := fizzBuzz(n)
		neg, err := fizzBuzz(-n)

		want := pos
		if pos == strconv.Itoa(n) {
			want = "-" + pos
		}
		if neg != want || err != nil {
			t.Fatalf("fizzBuzz(%d) = %q, %v; want %q, nil", -n, neg, err, want)
		}
	}
}
