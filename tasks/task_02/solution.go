package main

import "slices"

func rotateRunes(s string, shift int) string {
	runes := []rune(s)

	n := len(runes)
	if n == 0 {
		return s
	}
	
	shift = ((shift % n) + n) % n
	runes = slices.Concat(runes[shift:], runes[:shift])

	return string(runes)
}
