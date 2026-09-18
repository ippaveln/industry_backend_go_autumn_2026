package main

import (
	"math"
	"math/rand"
	"slices"
	"testing"
)

func TestDeltaStats(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []int64
		want Stats
	}{
		{name: "nil input", in: nil, want: Stats{}},
		{name: "empty input", in: []int64{}, want: Stats{}},
		{name: "single value", in: []int64{9}, want: Stats{}},
		{name: "mixed deltas", in: []int64{10, 13, 8, 8}, want: Stats{Count: 3, Sum: -2, Min: -5, Max: 3}},
		{name: "increasing", in: []int64{4, 9, 11}, want: Stats{Count: 2, Sum: 7, Min: 2, Max: 5}},
		{name: "decreasing", in: []int64{9, 4, 1}, want: Stats{Count: 2, Sum: -8, Min: -5, Max: -3}},
		{name: "zero delta", in: []int64{1, 1}, want: Stats{Count: 1, Sum: 0, Min: 0, Max: 0}},
		{name: "constant", in: []int64{-5, -5, -5, -5}, want: Stats{Count: 3, Sum: 0, Min: 0, Max: 0}},
		{name: "all deltas positive", in: []int64{5, 6, 8, 11}, want: Stats{Count: 3, Sum: 6, Min: 1, Max: 3}},
		{name: "all deltas negative", in: []int64{100, 0, -1}, want: Stats{Count: 2, Sum: -101, Min: -100, Max: -1}},
		{name: "extremes in the middle", in: []int64{0, 1, -9, 11, 12}, want: Stats{Count: 4, Sum: 12, Min: -10, Max: 20}},
		{name: "max int64 delta", in: []int64{0, math.MaxInt64}, want: Stats{Count: 1, Sum: math.MaxInt64, Min: math.MaxInt64, Max: math.MaxInt64}},
		{name: "min delta", in: []int64{math.MaxInt64, 0}, want: Stats{Count: 1, Sum: -math.MaxInt64, Min: -math.MaxInt64, Max: -math.MaxInt64}},
		{name: "from min int64", in: []int64{math.MinInt64 + 1, 0, 0}, want: Stats{Count: 2, Sum: math.MaxInt64, Min: 0, Max: math.MaxInt64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.in)

			got := Calc(tc.in)

			if got != tc.want {
				t.Errorf("Calc(%v) = %+v, want %+v", tc.in, got, tc.want)
			}
			if !slices.Equal(tc.in, before) {
				t.Errorf("input mutated: got %v, want %v", tc.in, before)
			}
		})
	}
}

func TestDeltaStats_Table(t *testing.T) {
	for _, tc := range []struct {
		in   []int64
		want Stats
	}{
		{[]int64{-571, 333}, Stats{Count: 1, Sum: 904, Min: 904, Max: 904}},
		{[]int64{552, -860, 793, 787, 808}, Stats{Count: 4, Sum: 256, Min: -1412, Max: 1653}},
		{[]int64{836, -230, 474, 43, 961, -212, 140, -426, 357, 561, 845}, Stats{Count: 10, Sum: 9, Min: -1173, Max: 918}},
		{[]int64{628385792567, -861484404902, 827659295000, -965095422599, 494877432240, 542510003035, -59385542467, 857521985237, -1068734998151, -837272035231}, Stats{Count: 9, Sum: -1465657827798, Min: -1926256983388, Max: 1689143699902}},
		{[]int64{-295, -755, 411}, Stats{Count: 2, Sum: 706, Min: -460, Max: 1166}},
		{[]int64{415, -360, -716, 123, 550, 873, -321, 713, 584, -932, 394}, Stats{Count: 10, Sum: -21, Min: -1516, Max: 1326}},
		{[]int64{340, 596, 925, -527, -863, 572, -284, -920, -72, -843, -863, -631}, Stats{Count: 11, Sum: -971, Min: -1452, Max: 1435}},
		{[]int64{-805021616032, 923818206804, -122445004206, 99924516496, 983347039361, -600536793513}, Stats{Count: 5, Sum: 204484822519, Min: -1583883832874, Max: 1728839822836}},
		{[]int64{-141, 444, 154, 635, 442, 849, 997, 296}, Stats{Count: 7, Sum: 437, Min: -701, Max: 585}},
		{[]int64{-274, -281, 751, 124, 931, -853, 811, -737, -329, 889, 748, -889}, Stats{Count: 11, Sum: -615, Min: -1784, Max: 1664}},
		{[]int64{-576, -9}, Stats{Count: 1, Sum: 567, Min: 567, Max: 567}},
		{[]int64{-169437779598, 319870445390, -97840463799, -906760770685, 795553492292, 524837425683, 481030307097, -95792316563, 269350137543, 144848547848, -1095239111043, -1089760793500}, Stats{Count: 11, Sum: -920323013902, Min: -1240087658891, Max: 1702314262977}},
		{[]int64{802, -528}, Stats{Count: 1, Sum: -1330, Min: -1330, Max: -1330}},
		{[]int64{613, -813, 223, 134, 785, -413, 262}, Stats{Count: 6, Sum: -351, Min: -1426, Max: 1036}},
		{[]int64{908, -559, 140, -425}, Stats{Count: 3, Sum: -1333, Min: -1467, Max: 699}},
		{[]int64{840131618234, -14712173726, 116189011122, -523052402892, -197098536506, -628934675220, -425604727290}, Stats{Count: 6, Sum: -1265736345524, Min: -854843791960, Max: 325953866386}},
		{[]int64{-340, 803, 95, -103, -427, 856, 825, 646, -875, -756, 250}, Stats{Count: 10, Sum: 590, Min: -1521, Max: 1283}},
		{[]int64{26, -208, 919, 473, -52}, Stats{Count: 4, Sum: -78, Min: -525, Max: 1127}},
		{[]int64{-572, -146, -325, -838, 746, 991, -33, 164, 574}, Stats{Count: 8, Sum: 1146, Min: -1024, Max: 1584}},
		{[]int64{-74880505116, 201848105202, 208570971837, 548823761801, -95332972473, 121281767996, 707423017865, 141132834121, -576286838149, 163755991095, -852856596214, -614961031149, 626079854392}, Stats{Count: 12, Sum: 700960359508, Min: -1016612587309, Max: 1241040885541}},
		{[]int64{-145, -794}, Stats{Count: 1, Sum: -649, Min: -649, Max: -649}},
		{[]int64{641, 496, 682, 145, 654, 533, 247, 719, 338, -62, -166}, Stats{Count: 10, Sum: -807, Min: -537, Max: 509}},
		{[]int64{-885, -704, 41, 943, -971, -792, -503, -680, 685, -254, 277, -507, 318}, Stats{Count: 12, Sum: 1203, Min: -1914, Max: 1365}},
		{[]int64{-359905507022, 919116553796, -267590667870, 518260508119, -984164579667, 320607617217, -836830346573, -603977040620, -655746407789, -689532782975}, Stats{Count: 9, Sum: -329627275953, Min: -1502425087786, Max: 1304772196884}},
		{[]int64{-591, -877, 190, -318, 401, 276, 478, -971, -908, 0}, Stats{Count: 9, Sum: 591, Min: -1449, Max: 1067}},
		{[]int64{608, 812, 362}, Stats{Count: 2, Sum: -246, Min: -450, Max: 204}},
		{[]int64{866, -4, 124, -637, -334, 674, 183, 234, -394, -445, 251, 382, 354}, Stats{Count: 12, Sum: -512, Min: -870, Max: 1008}},
		{[]int64{-600364121928, 418767970614}, Stats{Count: 1, Sum: 1019132092542, Min: 1019132092542, Max: 1019132092542}},
		{[]int64{366, 219, 562}, Stats{Count: 2, Sum: 196, Min: -147, Max: 343}},
		{[]int64{160, 527, 800, 871, -958, -868, 448, 845, 525, 727, 240}, Stats{Count: 10, Sum: 80, Min: -1829, Max: 1316}},
		{[]int64{805, -630}, Stats{Count: 1, Sum: -1435, Min: -1435, Max: -1435}},
		{[]int64{-41626217985, 293778261178}, Stats{Count: 1, Sum: 335404479163, Min: 335404479163, Max: 335404479163}},
		{[]int64{921, -331, -263, 961, -885, -909, -794, 909, -91, -891, -6}, Stats{Count: 10, Sum: -927, Min: -1846, Max: 1703}},
		{[]int64{319, 54, 919, 211, 480, 152, 373, 146, -399, 710}, Stats{Count: 9, Sum: 391, Min: -708, Max: 1109}},
		{[]int64{429, -32, -61}, Stats{Count: 2, Sum: -490, Min: -461, Max: -29}},
		{[]int64{1087371803761, -295382950847, 75773039519, -1085659065595, 752836610954, -184929420099, 248679067090}, Stats{Count: 6, Sum: -838692736671, Min: -1382754754608, Max: 1838495676549}},
		{[]int64{-379, -474, 106, -319, 615, 776, 656, -577, 659, 671, 881}, Stats{Count: 10, Sum: 1260, Min: -1233, Max: 1236}},
		{[]int64{150, 118, 78, 862, 812, -131, 250}, Stats{Count: 6, Sum: 100, Min: -943, Max: 784}},
		{[]int64{19, -892, 93, 216, 407, 810, -320, 967, -345, 315}, Stats{Count: 9, Sum: 296, Min: -1312, Max: 1287}},
		{[]int64{-362417707861, -1070659520130, 456986129365, -48210657071}, Stats{Count: 3, Sum: 314207050790, Min: -708241812269, Max: 1527645649495}},
	} {
		before := slices.Clone(tc.in)

		got := Calc(tc.in)

		if got != tc.want {
			t.Errorf("Calc(%v) = %+v, want %+v", tc.in, got, tc.want)
		}
		if !slices.Equal(tc.in, before) {
			t.Errorf("input mutated: got %v, want %v", tc.in, before)
		}
	}
}

func TestDeltaStats_NoAllocations(t *testing.T) {
	in := []int64{10, 13, 8, 8, -4, 20}

	allocs := testing.AllocsPerRun(100, func() { Calc(in) })

	if allocs != 0 {
		t.Fatalf("Calc allocates %.0f times per call, want 0 (O(1) extra memory)", allocs)
	}
}

// Shifting all values keeps the deltas; reversing negates them.
func TestDeltaStats_Invariants(t *testing.T) {
	r := rand.New(rand.NewSource(20261029))

	for n := 2; n < 200; n++ {
		in := make([]int64, n)
		for i := range in {
			in[i] = r.Int63n(2_000_001) - 1_000_000
		}
		got := Calc(in)

		if got.Count != n-1 {
			t.Fatalf("Calc(%v).Count = %d, want %d", in, got.Count, n-1)
		}
		if want := in[n-1] - in[0]; got.Sum != want {
			t.Fatalf("Calc(%v).Sum = %d, want last-first = %d", in, got.Sum, want)
		}
		if got.Min > got.Max || got.Min*int64(got.Count) > got.Sum || got.Max*int64(got.Count) < got.Sum {
			t.Fatalf("Calc(%v) = %+v: Min, Max and Sum are inconsistent", in, got)
		}

		shift := r.Int63n(1_000_000_000) - 500_000_000
		shifted := make([]int64, n)
		for i, v := range in {
			shifted[i] = v + shift
		}
		if s := Calc(shifted); s != got {
			t.Fatalf("Calc(%v) = %+v, want %+v as for the unshifted input", shifted, s, got)
		}

		reversed := slices.Clone(in)
		slices.Reverse(reversed)
		want := Stats{Count: got.Count, Sum: -got.Sum, Min: -got.Max, Max: -got.Min}
		if s := Calc(reversed); s != want {
			t.Fatalf("Calc(%v) = %+v, want %+v for the reversed input", reversed, s, want)
		}
	}
}
