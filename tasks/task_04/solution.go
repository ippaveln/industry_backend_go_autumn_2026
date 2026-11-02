package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) <= 1 {
		return Stats{}
	}
	stat := Stats{
		Min: nums[1] - nums[0],
		Max: nums[1] - nums[0],
		Sum: nums[1] - nums[0],
	}
	for i := 2; i < len(nums); i++ {
		n := nums[i] - nums[i-1]
		stat.Sum += n
		if stat.Max < n {
			stat.Max = n
		}
		if stat.Min > n {
			stat.Min = n
		}
	}
	stat.Count = len(nums) - 1
	return stat
}
