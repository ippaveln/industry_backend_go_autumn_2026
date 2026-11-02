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
	result := ""
	if n%3 == 0 {
		result = "Fizz"
	}
	if n%5 == 0 {
		result += "Buzz"
	}
	if result == "" {
		result = strconv.Itoa(n)
	}
	return result, nil
}
