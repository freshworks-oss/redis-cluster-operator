package utils

import "math"

func Round(num float64) int {
	switch {
	case math.IsNaN(num):
		return 0
	case math.IsInf(num, 1):
		return math.MaxInt
	case math.IsInf(num, -1):
		return math.MinInt
	}
	return int(num + math.Copysign(0.5, num))
}
