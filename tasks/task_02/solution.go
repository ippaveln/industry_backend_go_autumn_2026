package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	shift %= len(runes)
	if shift < 0 {
		shift += len(runes)
	}
	rotated := make([]rune, len(runes))
	n := copy(rotated, runes[shift:])
	copy(rotated[n:], runes[:shift])
	return string(rotated)
}
