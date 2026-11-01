package main

func rotateRunes(s string, shift int) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	n := len(runes)

	// Безопасная нормализация сдвига без переполнения int
	rem := shift % n
	k := (rem + n) % n

	resultAfterShift := make([]rune, n)
	for i := 0; i < n; i++ {
		newIndex := (i - k + n) % n
		resultAfterShift[newIndex] = runes[i]
	}

	return string(resultAfterShift)
}
