package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	if n <= 1 {
		return string(runes)
	}

	shift = (shift%n + n) % n
	if shift == 0 {
		return string(runes)
	}

	runes = append(runes[shift:], runes[:shift]...)

	return string(runes)
}
