package exchange

import (
	"context"
	"log"
	"time"

	"github.com/mwlazlo/srs/internal/pp"
)

type BarAggregator struct {
	duration time.Duration
	tickChan chan *pp.Tick
	barChan  chan *pp.Bar
	bar      *pp.Bar
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

func (b *BarAggregator) processTick(tick *pp.Tick) {
	if tick == nil && b.bar != nil {
		b.bar.CloseBar(nil)
		b.barChan <- b.bar
		return
	}
	if b.bar == nil {
		b.bar = pp.NewBar(b.duration)
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
			b.barChan <- b.bar
			b.bar = pp.NewBar(b.duration)
			b.bar.OpenBar(tick)
		} else {
			b.bar.AddTick(tick)
		}
	}
}

func (b *BarAggregator) Close() {
	close(b.tickChan)
}

func (b *BarAggregator) NextBar() (*pp.Bar, bool) {
	bar, ok := <-b.barChan
	return bar, ok
}

func (b *BarAggregator) AddTick(tick *pp.Tick) {
	b.tickChan <- tick
}

func NewBarAggregator(ctx context.Context, duration time.Duration) *BarAggregator {
	rv := &BarAggregator{
		duration: duration,
		tickChan: make(chan *pp.Tick),
		barChan:  make(chan *pp.Bar),
	}
	go rv.run(ctx)
	return rv
}
