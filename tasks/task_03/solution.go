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
	by3 := n%3 == 0
	by5 := n%5 == 0
	switch {
	case by3 && by5:
		return "FizzBuzz", nil
	case by3:
		return "Fizz", nil
	case by5:
		return "Buzz", nil
	}
	return strconv.Itoa(n), nil
}
