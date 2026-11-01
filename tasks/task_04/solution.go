package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	var result Stats

	for i := 1; i < len(nums); i++ {
		diff := nums[i] - nums[i-1]

		result.Count++
		result.Sum += diff

		if i == 1 {
			result.Min = diff
			result.Max = diff
			continue
		}

		if diff < result.Min {
			result.Min = diff
		}

		if diff > result.Max {
			result.Max = diff
		}
	}

	return result
}
