package main

func rotateRunes(s string, shift int) string {

	if s == "" {
		return s
	}

	runes := []rune(s)

	lenRunes := len(runes)

	shift = shift % lenRunes

	if shift < 0 {
		shift += lenRunes
	}

	shiftingRunes := append(runes[shift:], runes[:shift]...)
	return string(shiftingRunes)
}
