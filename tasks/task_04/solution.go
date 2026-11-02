package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	stats := Stats{}
	if len(nums) < 2 {
		return stats
	}
	for i := 1; i < len(nums); i++ {
		diff := nums[i] - nums[i-1]
		stats.Count++
		stats.Sum += diff
		if i == 1 || stats.Min > diff {
			stats.Min = diff
		}
		if i == 1 || stats.Max < diff {
			stats.Max = diff
		}
	}
	return stats
}
