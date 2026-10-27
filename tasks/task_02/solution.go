package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)

	if n == 0 {
		return ""
	}

	norm := shift % n

	if norm < 0 {
		norm += n
	}

	if norm == 0 {
		return string(runes)
	}

	return string(runes[norm:]) + string(runes[:norm])
}
