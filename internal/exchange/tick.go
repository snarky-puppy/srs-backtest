package exchange

import (
	"time"

	"github.com/mwlazlo/srs/internal"
)

type Tick struct {
	Timestamp time.Time
	Buy       float64
	Sell      float64
}

// MidPrice calculates the mid-point price between Buy and Sell.
func (t *Tick) MidPrice() float64 {
	return internal.Round4((t.Buy + t.Sell) / 2)
}

func (t *Tick) BaseTime(duration time.Duration) time.Time {
	return t.Timestamp.Truncate(duration)
}

func (t *Tick) DirectionPrice(direction Direction) float64 {
	switch direction {
	case Long:
		return t.Buy
	case Short:
		return t.Sell
	}
	return 0
}
