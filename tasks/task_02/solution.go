package main

import (
	"fmt"
)

func rotateRunes(s string, shift int) string {
	if len(s) == 0 {
		return ""
	}

	runeSlice := []rune(s)
	for shift < 0 {
		coefficient := -(shift / len(runeSlice))
		shift += (coefficient + 1) * len(runeSlice)
	}

	shift = shift % len(runeSlice)
	left, right := string(runeSlice[:shift]), string(runeSlice[shift:])
	return fmt.Sprintf("%s%s", right, left)
}
