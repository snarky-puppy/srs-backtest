package exchange

import "time"

type Bar struct {
	Open, High, Low, Close float64
	Timestamp              time.Time
	Duration               time.Duration
	lastPrice              float64
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

func NewBar(duration time.Duration) *Bar {
	return &Bar{Duration: duration}
}
