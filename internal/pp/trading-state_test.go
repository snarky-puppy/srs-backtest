package pp

import (
	"testing"
	"time"
)

func TestTradingStateString(t *testing.T) {
	testCases := []struct {
		state    TradingState
		expected string
	}{
		{Scanning, "Scanning"},
		{Trading, "Trading"},
		{Inactive, "Inactive"},
		{999, "Unknown state: 999"},
	}

	for _, tc := range testCases {
		actual := tc.state.String()
		if actual != tc.expected {
			t.Errorf("State.String() for state %d: expected %s, but got %s", tc.state, tc.expected, actual)
		}
	}
}

func TestNewTradingState(t *testing.T) {
	testCases := []struct {
		time     time.Time
		expected TradingState
	}{
		{time.Date(2023, time.May, 22, 7, 0, 0, 0, time.UTC), Inactive},
		{time.Date(2023, time.May, 22, 8, 0, 0, 0, time.UTC), Inactive},
		{time.Date(2023, time.May, 22, 8, 14, 59, 0, time.UTC), Inactive},
		{time.Date(2023, time.May, 22, 8, 15, 0, 0, time.UTC), Scanning},
		{time.Date(2023, time.May, 22, 8, 29, 59, 0, time.UTC), Scanning},
		{time.Date(2023, time.May, 22, 8, 30, 0, 0, time.UTC), Trading},
		{time.Date(2023, time.May, 22, 9, 0, 0, 0, time.UTC), Trading},
		{time.Date(2023, time.May, 22, 12, 0, 0, 0, time.UTC), Inactive},
	}

	for _, tc := range testCases {
		actual := NewTradingState(tc.time)
		if actual != tc.expected {
			t.Errorf("NewTradingState() for time %v: expected %s, but got %s", tc.time, tc.expected, actual)
		}
	}
}
