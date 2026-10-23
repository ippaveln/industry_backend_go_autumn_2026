package main

import (
	"errors"
	"strconv"
)

var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error) {
	if n == 0 {
		return "", ErrZero
	}

	div3, div5 := n%3 == 0, n%5 == 0
	switch {
	case div3 && div5:
		return "FizzBuzz", nil
	case div3:
		return "Fizz", nil
	case div5:
		return "Buzz", nil
	default:
		return strconv.Itoa(n), nil
	}
}
