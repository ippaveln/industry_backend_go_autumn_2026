package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	firstDiff := nums[1] - nums[0]
	stats := Stats{Count: len(nums) - 1, Min: firstDiff, Max: firstDiff}

	for i := 1; i < len(nums); i++ {
		diff := nums[i] - nums[i-1]

		stats.Sum += diff
		stats.Min = min(stats.Min, diff)
		stats.Max = max(stats.Max, diff)
	}

	return stats
}
