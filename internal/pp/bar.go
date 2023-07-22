package pp

import (
	"time"
)

type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Duration  time.Duration
	lastPrice float64
}

func (b *Bar) MarketCloseBar() bool {
	return b.Timestamp.Hour() == 17 && b.Timestamp.Minute() == 25
}

// Copy the bar
func (b *Bar) Copy() *Bar {
	return &Bar{
		Timestamp: b.Timestamp,
		Open:      b.Open,
		High:      b.High,
		Low:       b.Low,
		Close:     b.Close,
		Duration:  b.Duration,
	}
}

func (b *Bar) Add(b2 *Bar) {
	if b2.High > b.High {
		b.High = b2.High
	}
	if b2.Low < b.Low {
		b.Low = b2.Low
	}
	b.Close = b2.Close
}

// OpenBar sets the open price for the bar
func (b *Bar) OpenBar(t *Tick) {
	mid := t.MidPrice()
	b.Open = mid
	b.High = mid
	b.Low = mid
	b.Timestamp = t.BaseTime(b.Duration)
	b.lastPrice = mid
}

// CloseBar sets the close price for the bar
func (b *Bar) CloseBar(t *Tick) {
	if t != nil {
		b.Close = t.MidPrice()
	} else {
		b.Close = b.lastPrice
	}
}

// AddTick adds a new tick to the bar, updating the high and low prices as necessary.
func (b *Bar) AddTick(t *Tick) {
	midPrice := t.MidPrice()
	if midPrice > b.High || b.High == 0 {
		b.High = midPrice
	}
	if midPrice < b.Low || b.Low == 0 {
		b.Low = midPrice
	}
	b.lastPrice = midPrice
}

func (b *Bar) ClosingTime() time.Time {
	return b.Timestamp.Add(b.Duration)
}

func NewBar(duration time.Duration) *Bar {
	return &Bar{Duration: duration}
}
