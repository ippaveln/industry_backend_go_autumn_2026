package main

func rotateRunes(s string, shift int) string {
	rs := []rune(s)

	n := len(rs)
	if n == 0 {
		return ""
	}

	off := shift % n
	if off < 0 {
		off += n
	}

	rotated := make([]rune, 0, n)
	rotated = append(rotated, rs[off:]...)
	rotated = append(rotated, rs[:off]...)
	return string(rotated)
}
