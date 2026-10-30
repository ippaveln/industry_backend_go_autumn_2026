package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	var (
		count int   = 0
		sum   int64 = 0
		min         = nums[1] - nums[0]
		max         = min
	)

	for i := 1; i < len(nums); i++ {
		count++
		cur := nums[i] - nums[i-1]
		sum += cur

		if cur > max {
			max = cur
		}

		if cur < min {
			min = cur
		}
	}

	return Stats{count, sum, min, max}
}
