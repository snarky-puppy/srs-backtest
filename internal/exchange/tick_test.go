package exchange

import (
	"testing"
	"time"
)

func TestBaseTime(t *testing.T) {
	tick := Tick{Timestamp: time.Date(2023, 7, 13, 10, 7, 23, 0, time.UTC)}
	durations := []time.Duration{
		1 * time.Minute,
		2 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
	}

	expectedTimes := []time.Time{
		time.Date(2023, 7, 13, 10, 7, 0, 0, time.UTC), // nearest 1 minute
		time.Date(2023, 7, 13, 10, 6, 0, 0, time.UTC), // nearest 2 minutes
		time.Date(2023, 7, 13, 10, 5, 0, 0, time.UTC), // nearest 5 minutes
		time.Date(2023, 7, 13, 10, 0, 0, 0, time.UTC), // nearest 10 minutes
	}

	for i, d := range durations {
		result := tick.BaseTime(d)
		if !result.Equal(expectedTimes[i]) {
			t.Errorf("Expected BaseTime for duration %v to be %v, but got %v", d, expectedTimes[i], result)
		}
	}
}
