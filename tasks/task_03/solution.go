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

	byThree, byFive := n%3 == 0, n%5 == 0

	switch {
	case byThree && byFive:
		return "FizzBuzz", nil
	case byThree:
		return "Fizz", nil
	case byFive:
		return "Buzz", nil
	default:
		return strconv.Itoa(n), nil
	}
}
