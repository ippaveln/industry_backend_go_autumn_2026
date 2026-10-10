package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	first := nums[1] - nums[0]
	stats := Stats{Count: 1, Sum: first, Min: first, Max: first}
	for i := 2; i < len(nums); i++ {
		d := nums[i] - nums[i-1]
		stats.Count++
		stats.Sum += d
		stats.Min = min(stats.Min, d)
		stats.Max = max(stats.Max, d)
	}
	return stats
}
