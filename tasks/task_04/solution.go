package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	res := Stats{}
	if len(nums) < 2 {
		return res
	}
	res.Min = math.MaxInt64
	res.Max = math.MinInt64

	prev := nums[0]
	for _, current := range nums[1:] {
		diff := current - prev
		res.Add(diff)
		prev = current
	}
	return res
}

func (s *Stats) Add(diff int64) {
	s.Count += 1
	s.Sum += diff
	s.Min = min(diff, s.Min)
	s.Max = max(diff, s.Max)
}
