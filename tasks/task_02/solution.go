package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	
	if n < 1 {
		return s
	}

	shift %= n
	if shift < 0 {
		shift += n
	}

	return string(runes[shift:]) + string(runes[:shift])
}
