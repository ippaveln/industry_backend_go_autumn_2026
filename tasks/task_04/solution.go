package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {

	lenNums := len(nums)

	if lenNums < 2 {
		return Stats{}
	}

	var st Stats

	st.Min = math.MaxInt64
	st.Max = math.MinInt64

	st.Count = lenNums - 1

	for i := 1; i < lenNums; i++ {

		raz := nums[i] - nums[i-1]

		st.Sum += raz

		if raz < st.Min {
			st.Min = raz
		}

		if raz > st.Max {
			st.Max = raz
		}

	}

	return st
}
