package models

import (
	"testing"
)

func TestHistory_FindAverageHigh(t *testing.T) {
	h := History{
		bars: []*Bar{
			{High: 1},
			{High: 2},
			{High: 3},
			{High: 4},
			{High: 5},
		},
	}

	avg := h.FindAverageHigh(3)
	if avg != 4 {
		t.Errorf("expected %f, got %f", 4.0, avg)
	}
}

func TestHistory_FindAverageLow(t *testing.T) {
	h := History{
		bars: []*Bar{
			{Low: 1},
			{Low: 2},
			{Low: 3},
			{Low: 4},
			{Low: 5},
		},
	}

	avg := h.FindAverageLow(3)
	if avg != 4 {
		t.Errorf("expected %f, got %f", 4.0, avg)
	}
}
