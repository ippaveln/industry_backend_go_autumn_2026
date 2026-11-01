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

	fizz := n % 3 == 0
	buzz := n % 5 == 0
	
	if fizz && buzz {
		return "FizzBuzz", nil
	}
	if fizz {
		return "Fizz", nil
	}
	if buzz {
		return "Buzz", nil
	}

	return strconv.Itoa(n), nil
}
