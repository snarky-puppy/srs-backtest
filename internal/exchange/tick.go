package exchange

import (
	"time"
)

type Tick struct {
	Timestamp time.Time
	Buy       float64
	Sell      float64
}

// MidPrice calculates the mid-point price between Buy and Sell.
func (t *Tick) MidPrice() float64 {
	return (t.Buy + t.Sell) / 2
}

func (t *Tick) BaseTime(duration time.Duration) time.Time {
	return t.Timestamp.Truncate(duration)
}
