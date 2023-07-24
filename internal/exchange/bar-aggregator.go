package exchange

import (
	"log"
	"time"
)

type BarAggregator struct {
	duration time.Duration
	bar      *Bar
}

func (b *BarAggregator) ProcessTick(tick *Tick) (rv *Bar) {
	if tick == nil && b.bar != nil {
		b.bar.CloseBar(nil)
		rv = b.bar
		return
	}
	if b.bar == nil {
		b.bar = NewBar(b.duration)
		b.bar.OpenBar(tick)
	} else {
		// if the tick is before the current bar, log and ignore
		if tick.BaseTime(b.duration).Before(b.bar.Timestamp) {
			log.Printf("Tick %v is before current bar %v", tick, b.bar)
			return
		}
		// if the tick is outside the current bar, close the bar and start a new one
		if tick.BaseTime(b.duration) != b.bar.Timestamp {
			b.bar.CloseBar(tick)
			rv = b.bar
			b.bar = NewBar(b.duration)
			b.bar.OpenBar(tick)
		} else {
			b.bar.AddTick(tick)
		}
	}
	return
}

func NewBarAggregator(duration time.Duration) *BarAggregator {
	rv := &BarAggregator{
		duration: duration,
	}
	return rv
}
