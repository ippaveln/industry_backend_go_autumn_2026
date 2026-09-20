package main

import (
	"math/rand"
	"strings"
	"testing"
)

func TestRotate(t *testing.T) {
	for _, c := range []struct {
		s    string
		k    int
		want string
	}{{"", 5, ""}, {"abc", 0, "abc"}, {"abc", 1, "bca"}, {"А🙂Б", -1, "БА🙂"}, {"А🙂Б", 4, "🙂БА"}, {"e\u0301x", 1, "\u0301xe"}, {"\xffa", 0, "�a"}, {"x", -int(^uint(0)>>1) - 1, "x"}} {
		if g := rotateRunes(c.s, c.k); g != c.want {
			t.Errorf("%q,%d: %q != %q", c.s, c.k, g, c.want)
		}
	}
}
func TestRotateModel(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	alphabet := []rune("аб🙂界é")
	for n := 1; n < 60; n++ {
		a := make([]rune, n)
		for i := range a {
			a[i] = alphabet[r.Intn(len(alphabet))]
		}
		for _, k := range []int{r.Intn(1000) - 500, -int(^uint(0)>>1) - 1, int(^uint(0) >> 1)} {
			var b strings.Builder
			for j := 0; j < n; j++ {
				idx := (j + k%n + n) % n
				b.WriteRune(a[idx])
			}
			if g := rotateRunes(string(a), k); g != b.String() {
				t.Fatalf("n=%d k=%d", n, k)
			}
		}
	}
}
