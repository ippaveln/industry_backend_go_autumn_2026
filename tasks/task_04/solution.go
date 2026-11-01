package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	delt := nums[1] - nums[0]
	st := Stats{
		Count: 1,
		Sum:   delt,
		Min:   delt,
		Max:   delt,
	}

	for i := 2; i < len(nums); i++ {
		delt = nums[i] - nums[i-1]
		st.Count++
		st.Sum += delt

		if delt < st.Min {
			st.Min = delt
		}

		if delt > st.Max {
			st.Max = delt
		}
	}
	return st
}
