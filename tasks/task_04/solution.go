package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	value := nums[1] - nums[0]

	stats := Stats{
		Count: 1,
		Sum:   value, Min: value, Max: value,
	}

	for i := 2; i < len(nums); i++ {
		stats.Count++
		value = nums[i] - nums[i-1]
		stats.Sum += value
		stats.Min = min(stats.Min, value)
		stats.Max = max(stats.Max, value)
	}
	return stats
}
