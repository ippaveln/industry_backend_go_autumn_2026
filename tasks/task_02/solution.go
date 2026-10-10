package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}

	k := shift % n
	if k < 0 {
		k += n
	}

	rotated := make([]rune, 0, n)
	rotated = append(rotated, runes[k:]...)
	rotated = append(rotated, runes[:k]...)
	return string(rotated)
}
