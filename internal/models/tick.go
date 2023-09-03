package models

import (
	"time"

	"github.com/mwlazlo/srs/internal"
)

type Tick struct {
	Symbol    Symbol
	Timestamp time.Time
	Bid       float64
	Ask       float64
	Key       string
}

// MidPrice calculates the mid-point price between Bid and Ask.
func (t *Tick) MidPrice() float64 {
	return internal.Round4((t.Bid + t.Ask) / 2)
}

func (t *Tick) BaseTime(duration time.Duration) time.Time {
	return t.Timestamp.Truncate(duration)
}

func (t *Tick) DirectionPrice(direction Direction) float64 {
	switch direction {
	case Long:
		return t.Bid
	case Short:
		return t.Ask
	}
	return 0
}
