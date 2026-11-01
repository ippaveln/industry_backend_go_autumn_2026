package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	stats := Stats{
		Count: 1,
		Sum: nums[1] - nums[0],
		Min: nums[1] - nums[0],
		Max: nums[1] - nums[0],
	}

	for i:= 2; i < len(nums); i++ {
		diff := nums[i] - nums[i - 1]
		stats.Count++
		stats.Sum += diff

		if diff < stats.Min {
			stats.Min = diff
		}
		if diff > stats.Max {
			stats.Max = diff
		}	
	}

	return stats	
}
