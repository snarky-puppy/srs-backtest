package exchange

import (
	"context"
	"time"
)

type BarAggregator struct {
	duration time.Duration
	tickChan chan *Tick
	barChan  chan *Bar
	bar      *Bar
}

func (b *BarAggregator) run(ctx context.Context) {
	for {
		select {
		case tick := <-b.tickChan:
			b.processTick(tick)
			if tick == nil {
				close(b.barChan)
				return
			}
		}
	}
}

func (b *BarAggregator) processTick(tick *Tick) {
	if tick == nil && b.bar != nil {
		b.bar.CloseBar(nil)
		b.barChan <- b.bar
		return
	}
	if b.bar == nil {
		b.bar = NewBar(b.duration)
		b.bar.OpenBar(tick)
	} else {
		// if the tick is outside the current bar, close the bar and start a new one
		if tick.BaseTime(b.duration) != b.bar.Timestamp {
			b.bar.CloseBar(tick)
			b.barChan <- b.bar
			b.bar = NewBar(b.duration)
			b.bar.OpenBar(tick)
		} else {
			b.bar.AddTick(tick)
		}
	}
}

func (b *BarAggregator) Close() {
	close(b.tickChan)
}

func (b *BarAggregator) NextBar() (*Bar, bool) {
	bar, ok := <-b.barChan
	return bar, ok
}

func (b *BarAggregator) AddTick(tick *Tick) {
	b.tickChan <- tick
}

func NewBarAggregator(ctx context.Context, duration time.Duration) *BarAggregator {
	rv := &BarAggregator{
		duration: duration,
		tickChan: make(chan *Tick),
		barChan:  make(chan *Bar),
	}
	go rv.run(ctx)
	return rv
}
