package pp

import (
	"math"
)

// Refactored Atr and Sma function
func Atr(period int, bars Series) ([]float64, []float64) {
	tr := make([]float64, len(bars))

	for i := 0; i < len(tr); i++ {
		tr[i] = math.Max(bars[i].High-bars[i].Low, math.Max(bars[i].High-bars[i].Close, bars[i].Close-bars[i].Low))
	}

	atr := Sma(period, tr)

	return tr, atr
}

func Sma(period int, values []float64) []float64 {
	result := make([]float64, len(values))
	sum := float64(0)

	for i, value := range values {
		count := i + 1
		sum += value

		if i >= period {
			sum -= values[i-period]
			count = period
		}

		result[i] = sum / float64(count)
	}

	return result
}
