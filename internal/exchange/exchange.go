package exchange

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/mwlazlo/srs/internal/pp"
)

type Strategy interface {
	FiveMinBar(bar *pp.Bar)
	TradeStoppedOut(trade *pp.Trade)
	TradeOpened(trade *pp.Trade)
	SetExchange(exch *Exchange)
}

type Exchange struct {
	reader     *pp.TickReader
	aggregator *BarAggregator
	Trades     *TradeManager
	strategy   Strategy
	done       chan struct{}
}

func NewExchange(ctx context.Context, file string, s Strategy) *Exchange {
	reader, err := pp.NewTickReader(file)
	if err != nil {
		panic(err)
	}
	rv := &Exchange{
		reader:     reader,
		strategy:   s,
		done:       make(chan struct{}),
		aggregator: NewBarAggregator(ctx, time.Minute*5),
		Trades:     NewTradeManager(ctx, s),
	}
	go rv.tickReader(ctx)
	go rv.fiveMinBarReader()
	s.SetExchange(rv)
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
			e.Trades.addTick(tick)
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
