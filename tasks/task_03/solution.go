package main

import (
	"errors"
	"strconv"
)

var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error) {
	res := ""
	if n == 0 {
		return res, ErrZero
	}
	if n%3 == 0 {
		res += "Fizz"
	}
	if n%5 == 0 {
		res += "Buzz"
	}
	if res != "" {
		return res, nil
	} else {
		return strconv.Itoa(n), nil
	}
}
