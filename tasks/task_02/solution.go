package main

func rotateRunes(s string, shift int) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	shift = shift % len(r)
	if shift < 0 {
		shift += len(r)
	}
	new_r := make([]rune, 0, len(r))
	for i := shift; i < len(r); i++ {
		new_r = append(new_r, r[i])
	}
	for i := 0; i < shift; i++ {
		new_r = append(new_r, r[i])
	}

	return string(new_r)
}
