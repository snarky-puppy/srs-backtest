package exchange

import (
	"time"

	"github.com/mwlazlo/srs/internal/pp"
)

type TradeCloseReason int

const (
	StopLossHit TradeCloseReason = iota
	ProfitTargetHit
	TradeClosedManually
)

func (r TradeCloseReason) String() string {
	switch r {
	case StopLossHit:
		return "StopLossHit"
	case ProfitTargetHit:
		return "ProfitTargetHit"
	case TradeClosedManually:
		return "TradeClosedManually"
	}
	return "Unknown"
}

type EntryScanner = func(history *pp.History, tradeManager *TradeManager)

type exchange interface {
	CreateOrder(trade *ExTrade) *ExTrade
}

type TradeManager struct {
	aggregator    *BarAggregator
	exchange      exchange
	entryScanners []EntryScanner
	orders        []*pp.Trade
	positions     []*pp.Trade
}

func (t *TradeManager) PositionClosed(trade *ExTrade) {
	//TODO implement me
	panic("implement me")
}

func (t *TradeManager) PositionOpened(trade *ExTrade) {
	//TODO implement me
	panic("implement me")
}

func (t *TradeManager) HandleTick(tick *pp.Tick) {
	bar := t.aggregator.processTick(tick)
	if bar != nil {
		for _, scanner := range t.entryScanners {
			scanner(bar.History, t)
		}
	}
}

func NewTradeManager(scanners ...EntryScanner) *TradeManager {
	rv := &TradeManager{
		aggregator:    NewBarAggregator(time.Minute * 5),
		entryScanners: scanners,
	}
	return rv
}

func (t *TradeManager) SetExchange(exchange exchange) {
	t.exchange = exchange
}

func (t *TradeManager) AddSignal(signal *pp.Signal) {
	if t.activeSignal != nil {
		panic("active signal already exists")
	}
	t.activeSignal = signal
	t.history.AddSignal(signal)
}

func (t *TradeManager) CreateOrder(trade *pp.Trade) {

}
