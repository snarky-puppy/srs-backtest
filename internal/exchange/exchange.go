package exchange

import (
	"context"
	"errors"
	"io"
	"time"
)

type Strategy interface {
	FiveMinBar(bar *Bar)
}

type Trade struct {
	OpenTime  time.Time
	OpenPrice float64

	StopPrice float64
	StopTime  time.Time

	Qty float64
}

type Exchange struct {
	reader     *TickReader
	aggregator *BarAggregator
	strategy   Strategy
	openTrades []*Trade
	done       chan struct{}
}

func NewExchange(ctx context.Context, file string, s Strategy) *Exchange {
	reader, err := NewTickReader(file)
	if err != nil {
		panic(err)
	}
	rv := &Exchange{
		reader:     reader,
		strategy:   s,
		done:       make(chan struct{}),
		aggregator: NewBarAggregator(ctx, time.Minute*5),
	}
	go rv.tickReader(ctx)
	go rv.fiveMinBarReader()
	return rv
}

func (e *Exchange) Wait() {
	<-e.done
}

func (e *Exchange) tickReader(ctx context.Context) {
	defer close(e.done)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			tick, err := e.reader.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				panic(err)
			}
			e.aggregator.AddTick(tick)
		}
	}
}

func (e *Exchange) fiveMinBarReader() {
	for {
		bar, ok := e.aggregator.NextBar()
		if !ok {
			return
		}
		e.strategy.FiveMinBar(bar)
	}
}
