package main

func rotateRunes(s string, shift int) string {
	rune := []rune(s)
	if len(rune) == 0 {
		return ""
	}

	of := shift % len(rune)
	if of < 0 {
		of += len(rune)
	}
	return string(rune[of:]) + string(rune[:of])
}
