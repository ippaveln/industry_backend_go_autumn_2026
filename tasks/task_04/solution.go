package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	// Первая разность задаёт начальные Min и Max: нейтральных значений
	// вроде нуля здесь нет, а math.MaxInt64 как «пустой минимум» лишний,
	// потому что разность хотя бы одна точно есть.
	first := nums[1] - nums[0]
	stats := Stats{Count: len(nums) - 1, Sum: first, Min: first, Max: first}

	for i := 2; i < len(nums); i++ {
		delta := nums[i] - nums[i-1]
		stats.Sum += delta
		stats.Min = min(stats.Min, delta)
		stats.Max = max(stats.Max, delta)
	}

	return stats
}
