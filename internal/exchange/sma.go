package exchange

import "math"

type SMA struct {
	Period  int
	Values  []float64
	History []float64
}

func (s *SMA) AddBar(bar *Bar) {
	s.Values = append(s.Values, bar.AvgPrice())

	// If the length of values is larger than the period, remove the oldest value.
	if len(s.Values) > s.Period {
		s.Values = s.Values[1:]
	}

	// calculate and add the current SMA value to the history
	smaValue := s.Calculate()
	s.History = append(s.History, smaValue)

	// If the length of history is larger than the period, remove the oldest value.
	if len(s.History) > 100 {
		s.History = s.History[1:]
	}
}

func (s *SMA) Calculate() float64 {
	sum := 0.0
	for _, v := range s.Values {
		sum += v
	}

	// If there are no values, return 0.
	if len(s.Values) == 0 {
		return 0
	}

	return sum / float64(len(s.Values))
}

func (s *SMA) GetAngle(n int) float64 {
	if len(s.History) < 2 || n < 2 {
		return 0
	}

	startIndex := len(s.History) - n
	if startIndex < 0 {
		startIndex = 0
	}

	// Calculate the difference in SMA values and time (assuming time steps of 1)
	yDiff := s.History[len(s.History)-1] - s.History[startIndex]
	xDiff := float64(len(s.History) - 1 - startIndex)

	// Calculate the angle in radians
	angleRad := math.Atan2(yDiff, xDiff)

	// Convert the angle to degrees
	angleDeg := angleRad * (180.0 / math.Pi)

	return angleDeg
}

func (s *SMA) CrossedOver(sma *SMA, n int) bool {
	if len(s.History) < n || len(sma.History) < n {
		return false
	}

	// get the last n values from both SMAs
	sLastN := s.History[len(s.History)-n:]
	smaLastN := sma.History[len(sma.History)-n:]

	// if s was initially below sma, and is now above, return true
	if sLastN[0] < smaLastN[0] && sLastN[n-1] > smaLastN[n-1] {
		return true
	}

	return false
}

func (s *SMA) CrossedUnder(sma *SMA, n int) bool {
	if len(s.History) < n || len(sma.History) < n {
		return false
	}

	// get the last n values from both SMAs
	sLastN := s.History[len(s.History)-n:]
	smaLastN := sma.History[len(sma.History)-n:]

	// if s was initially above sma, and is now below, return true
	if sLastN[0] > smaLastN[0] && sLastN[n-1] < smaLastN[n-1] {
		return true
	}

	return false
}
