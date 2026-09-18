package main

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

func TestRotate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		s     string
		shift int
		want  string
	}{
		{name: "empty string", s: "", shift: 5, want: ""},
		{name: "empty string, min int", s: "", shift: math.MinInt, want: ""},
		{name: "empty string, max int", s: "", shift: math.MaxInt, want: ""},
		{name: "zero shift", s: "abc", shift: 0, want: "abc"},
		{name: "shift left by one", s: "abc", shift: 1, want: "bca"},
		{name: "shift left by two", s: "abc", shift: 2, want: "cab"},
		{name: "readme example left", s: "А🙂Б", shift: 1, want: "🙂БА"},
		{name: "negative shift", s: "А🙂Б", shift: -1, want: "БА🙂"},
		{name: "shift equal to length", s: "А🙂Б", shift: 3, want: "А🙂Б"},
		{name: "shift beyond length", s: "А🙂Б", shift: 4, want: "🙂БА"},
		{name: "negative shift beyond length", s: "А🙂Б", shift: -4, want: "БА🙂"},
		{name: "combining mark is a rune", s: "éx", shift: 1, want: "́xe"},
		{name: "invalid UTF-8 replaced", s: "\xffa", shift: 0, want: "�a"},
		{name: "invalid UTF-8 rotated", s: "\xffa", shift: 1, want: "a�"},
		{name: "truncated sequence", s: "a\xe2\x82", shift: 1, want: "��a"},
		{name: "min int shift", s: "x", shift: math.MinInt, want: "x"},
		{name: "max int shift", s: "x", shift: math.MaxInt, want: "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rotateRunes(tc.s, tc.shift)

			if got != tc.want {
				t.Errorf("rotateRunes(%q, %d) = %q, want %q", tc.s, tc.shift, got, tc.want)
			}
		})
	}
}

func TestRotate_Table(t *testing.T) {
	for _, tc := range []struct {
		s     string
		shift int
		want  string
	}{
		{"a", 1, "a"},
		{"a", -1, "a"},
		{"a", 0, "a"},
		{"a", 2, "a"},
		{"a", -2, "a"},
		{"a", 3, "a"},
		{"ab", 1, "ba"},
		{"ab", -1, "ba"},
		{"ab", 2, "ab"},
		{"ab", 3, "ba"},
		{"ab", -3, "ba"},
		{"ab", 5, "ba"},
		{"abc", 1, "bca"},
		{"abc", -1, "cab"},
		{"abc", 2, "cab"},
		{"abc", 3, "abc"},
		{"abc", 4, "bca"},
		{"abc", -4, "cab"},
		{"abc", 7, "bca"},
		{"hello", 1, "elloh"},
		{"hello", -1, "ohell"},
		{"hello", 4, "ohell"},
		{"hello", 5, "hello"},
		{"hello", 6, "elloh"},
		{"hello", -6, "ohell"},
		{"hello", 11, "elloh"},
		{"привет", 1, "риветп"},
		{"привет", -1, "тприве"},
		{"привет", 5, "тприве"},
		{"привет", 6, "привет"},
		{"привет", 7, "риветп"},
		{"привет", -7, "тприве"},
		{"привет", 13, "риветп"},
		{"Го🙂", 1, "о🙂Г"},
		{"Го🙂", -1, "🙂Го"},
		{"Го🙂", 2, "🙂Го"},
		{"Го🙂", 3, "Го🙂"},
		{"Го🙂", 4, "о🙂Г"},
		{"Го🙂", -4, "🙂Го"},
		{"Го🙂", 7, "о🙂Г"},
		{"日本語テキスト", 1, "本語テキスト日"},
		{"日本語テキスト", -1, "ト日本語テキス"},
		{"日本語テキスト", 6, "ト日本語テキス"},
		{"日本語テキスト", 7, "日本語テキスト"},
		{"日本語テキスト", 8, "本語テキスト日"},
		{"日本語テキスト", -8, "ト日本語テキス"},
		{"日本語テキスト", 15, "本語テキスト日"},
		{"éa", 1, "́ae"},
		{"éa", -1, "aé"},
		{"éa", 2, "aé"},
		{"éa", 3, "éa"},
		{"éa", 4, "́ae"},
		{"éa", -4, "aé"},
		{"éa", 7, "́ae"},
		{"👨\u200d👩\u200d👧", 1, "\u200d👩\u200d👧👨"},
		{"👨\u200d👩\u200d👧", -1, "👧👨\u200d👩\u200d"},
		{"👨\u200d👩\u200d👧", 4, "👧👨\u200d👩\u200d"},
		{"👨\u200d👩\u200d👧", 5, "👨\u200d👩\u200d👧"},
		{"👨\u200d👩\u200d👧", 6, "\u200d👩\u200d👧👨"},
		{"👨\u200d👩\u200d👧", -6, "👧👨\u200d👩\u200d"},
		{"👨\u200d👩\u200d👧", 11, "\u200d👩\u200d👧👨"},
		{"\xff\xfe", 1, "��"},
		{"\xff\xfe", -1, "��"},
		{"\xff\xfe", 2, "��"},
		{"\xff\xfe", 3, "��"},
		{"\xff\xfe", -3, "��"},
		{"\xff\xfe", 5, "��"},
		{"a\xc3b", 1, "�ba"},
		{"a\xc3b", -1, "ba�"},
		{"a\xc3b", 2, "ba�"},
		{"a\xc3b", 3, "a�b"},
		{"a\xc3b", 4, "�ba"},
		{"a\xc3b", -4, "ba�"},
		{"a\xc3b", 7, "�ba"},
		{"€€", 1, "€€"},
		{"€€", -1, "€€"},
		{"€€", 2, "€€"},
		{"€€", 3, "€€"},
		{"€€", -3, "€€"},
		{"€€", 5, "€€"},
		{"🙂\xff\xc3界\xf0\x9f🚀\xe2\x82\xc3🚀\u200d\xe2\x82\xe2\x82zz🚀 é", 179, "界��🚀���🚀\u200d����zz🚀 é🙂��"},
		{"\xffЯжé0語é🙂ё\xc30b 🚀 0\xffЯ🙂界\u200d語", math.MaxInt, "é🙂ё�0b 🚀 0�Я🙂界\u200d語�Яжé0語"},
		{"\xc3z béж0語\u200d", math.MinInt, " béж0語\u200d�z"},
		{"ж🚀ж \xe2\x82ж\u200d🚀🙂ЯЯzbжёa語Я\u200d", 342, "ж ��ж\u200d🚀🙂ЯЯzbжёa語Я\u200dж🚀"},
		{"0b\xffёa\xe2\x82Я語語🙂ж\xe2\x82ё Я🙂b \u200d\xe2\x82🚀🚀Я", math.MinInt, "b \u200d��🚀🚀Я0b�ёa��Я語語🙂ж��ё Я🙂"},
		{"Я\xe2\x82🙂жé\xc3b\xc3b\xf0\x9f🚀z\xc3\xe2\x82a\xff語\u200d\xe2\x82", math.MinInt, "��a�語\u200d��Я��🙂жé�b�b��🚀z�"},
		{"\xc3Я0z0\xffЯж0語ж0\xe2\x82b🚀b é🚀", math.MaxInt, "ж0語ж0��b🚀b é🚀�Я0z0�Я"},
		{"\u200déb語語", math.MaxInt - 1, "\u200déb語語"},
		{"ж🙂zzёё\xe2\x82azёЯ語\xe2\x82\xc3🙂ё界\xc3Я\xf0\x9f", math.MinInt + 1, "��ж🙂zzёё��azёЯ語���🙂ё界�Я"},
		{"🚀0é\xc3", math.MinInt, "é�🚀0"},
		{"\xe2\x820ёa\xf0\x9f語語z\xf0\x9fb ёё🙂\xe2\x82\xf0\x9f0a\xff界🚀\xc3ё\xf0\x9f", math.MinInt, "a�界🚀�ё����0ёa��語語z��b ёё🙂����0"},
		{"\xe2\x82bжb é\xffb\u200daж ж界\xc3\u200dz ", math.MinInt, "ж ж界�\u200dz ��bжb é�b\u200da"},
		{" 🙂🙂azéa\xc3b\xc3\xe2\x82", math.MaxInt - 1, "́a�b��� 🙂🙂aze"},
		{"\xc3\xc3🙂aé界🚀\xe2\x82界\xff\u200d\u200déé", 179, "́��🙂aé界🚀��界�\u200d\u200dée"},
		{"語a\xff🙂ézж🙂é\u200déz界0界жaёb界🚀a", -879, "b界🚀a語a�🙂ézж🙂é\u200déz界0界жaё"},
		{"\xf0\x9f語b語\u200db", math.MinInt + 1, "��語b語\u200db"},
		{"0語🙂a🙂\u200dё語 0", 235, "\u200dё語 00語🙂a🙂"},
		{"\xf0\x9fb🚀界\xff🚀", math.MinInt + 1, "��b🚀界�🚀"},
		{"0語\u200d\u200d\xf0\x9f\xff\u200d🙂🙂\xffbz界 \xc3🚀Я éz\xf0\x9f\xffё🙂", math.MaxInt, "ё🙂0語\u200d\u200d���\u200d🙂🙂�bz界 �🚀Я éz���"},
		{"жЯ🙂🚀\xffё\xf0\x9fё 0", 167, "🙂🚀�ё��ё 0жЯ"},
		{"\u200d ж語b\xf0\x9f\xff\xe2\x82", math.MaxInt - 1, "����\u200d ж語b�"},
		{" \xffжé\xf0\x9fz", math.MaxInt - 1, "�z �жé�"},
		{"\xe2\x82語ёж\u200dё🚀\xf0\x9f🚀é\xf0\x9f\xc3🙂a", math.MaxInt, "🚀��🚀é���🙂a��語ёж\u200dё"},
		{"\xe2\x820 🚀ё\xe2\x82ж\u200d🚀語b", -642, "ж\u200d🚀語b��0 🚀ё��"},
		{"zzé語\xe2\x82zж語\xffЯ語🚀\xe2\x82語語b\u200dz🚀00é\u200d", math.MinInt + 1, "é語��zж語�Я語🚀��語語b\u200dz🚀00é\u200dzz"},
		{" \xc3Я\xe2\x82🚀 z\u200déb🙂語語界", -393, "z\u200déb🙂語語界 �Я��🚀 "},
		{"語🙂Яё0Яё0界a🚀語Я", -701, "🙂Яё0Яё0界a🚀語Я語"},
		{"\xc3ж 🚀a b 語ё 🚀b🙂", 585, "🚀b🙂�ж 🚀a b 語ё "},
		{"語é\xc3\xff", math.MaxInt - 1, "é��語"},
		{"界0\u200d\xc3bжЯa\xf0\x9f\xf0\x9f界Яa\u200dbжéжЯ\u200d", math.MaxInt - 1, "0\u200d�bжЯa����界Яa\u200dbжéжЯ\u200d界"},
		{"ё界0\xc3", -655, "界0�ё"},
		{"🙂\xf0\x9f語\xc3界0ёЯ\xfféaz\u200d", math.MaxInt, "ёЯ�éaz\u200d🙂��語�界0"},
		{"a🙂\xc3z\xf0\x9fb0界\xf0\x9fЯz\xe2\x82Я ж\xc3Яz 界\xc3\u200d\u200da", -346, "�b0界��Яz��Я ж�Яz 界�\u200d\u200daa🙂�z�"},
		{"z", math.MaxInt - 1, "z"},
		{"\xff語\u200d b \xc3\xe2\x82é", math.MaxInt - 1, "���é�語\u200d b "},
		{"語ézéё\u200d語zжёжz🙂é界🚀Яbb🚀zЯa\xf0\x9f", 687, "é界🚀Яbb🚀zЯa��語ézéё\u200d語zжёжz🙂"},
		{"\u200d語\xf0\x9f🙂🚀 Я界 語🙂a0a🚀00界", math.MaxInt, "0界\u200d語��🙂🚀 Я界 語🙂a0a🚀0"},
		{"界\xc3b\xf0\x9f🙂界ж🙂\xf0\x9faé界ж", -156, "�🙂界ж🙂��aé界ж界�b�"},
		{"\xc3Я語🙂\xe2\x82\xe2\x82\xf0\x9f0\u200d🚀 b🙂\xe2\x82\xc3\xc3🚀a", math.MaxInt - 1, "����0\u200d🚀 b🙂����🚀a�Я語🙂��"},
		{"🚀\u200dжё界éЯb界\xff0\xf0\x9féa🚀🚀Яb", -71, "�éa🚀🚀Яb🚀\u200dжё界éЯb界�0�"},
	} {
		got := rotateRunes(tc.s, tc.shift)

		if got != tc.want {
			t.Errorf("rotateRunes(%q, %d) = %q, want %q", tc.s, tc.shift, got, tc.want)
		}
	}
}

func TestRotate_Blocks(t *testing.T) {
	for _, n := range []int{1, 2, 7, 1000, 100_000} {
		left, right := strings.Repeat("я", n), strings.Repeat("🙂", 2*n)
		s := left + right

		if got := rotateRunes(s, n); got != right+left {
			t.Fatalf("n=%d: rotating left by n must move the first block to the end", n)
		}
		if got := rotateRunes(s, -2*n); got != right+left {
			t.Fatalf("n=%d: rotating right by 2n must move the last block to the front", n)
		}
		if got := rotateRunes(s, 3*n); got != s {
			t.Fatalf("n=%d: rotating by the length must keep the string", n)
		}
	}
}

func TestRotate_InverseAndComposition(t *testing.T) {
	r := rand.New(rand.NewSource(20261029))
	alphabet := []rune("ab界🙂жé́")

	for n := 1; n <= 40; n++ {
		runes := make([]rune, n)
		for i := range runes {
			runes[i] = alphabet[r.Intn(len(alphabet))]
		}
		s := string(runes)
		a, b := r.Intn(200)-100, r.Intn(200)-100

		if got := rotateRunes(rotateRunes(s, a), -a); got != s {
			t.Fatalf("rotateRunes(rotateRunes(%q, %d), %d) = %q, want the original string", s, a, -a, got)
		}
		if got, want := rotateRunes(rotateRunes(s, a), b), rotateRunes(s, a+b); got != want {
			t.Fatalf("rotating %q by %d then %d = %q, rotating by %d = %q", s, a, b, got, a+b, want)
		}
		if got, want := rotateRunes(s, math.MinInt), rotateRunes(s, math.MinInt%n); got != want {
			t.Fatalf("rotateRunes(%q, MinInt) = %q, want %q (same as shift %d)", s, got, want, math.MinInt%n)
		}
		if got, want := rotateRunes(s, math.MaxInt), rotateRunes(s, math.MaxInt%n); got != want {
			t.Fatalf("rotateRunes(%q, MaxInt) = %q, want %q (same as shift %d)", s, got, want, math.MaxInt%n)
		}
	}
}
