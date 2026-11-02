package main

func rotateRunes(s string, shift int) string {
	runedString := []rune(s)
	if len(runedString) == 0 {
		return ""
	}
	n := len(runedString)
	normalizedShift := ((shift % n) + n) % n
	newString := make([]rune, n)
	for i := range runedString {
		oldIndex := (i + normalizedShift) % n
		newString[i] = runedString[oldIndex]
	}
	return string(newString)
}
