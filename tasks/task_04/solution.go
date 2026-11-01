package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) (stats Stats) {
	if len(nums) < 2 {
		return
	}

	stats.Min = math.MaxInt64
	stats.Max = math.MinInt64
	var diff int64
	for i := 1; i < len(nums); i++ {
		stats.Count++
		diff = nums[i] - nums[i-1]
		stats.Sum += diff
		if diff > stats.Max {
			stats.Max = diff
		}
		if diff < stats.Min {
			stats.Min = diff
		}
	}

	return
}
