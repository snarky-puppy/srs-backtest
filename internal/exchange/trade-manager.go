package exchange

import (
	"log"
	"time"
)

const (
	ScanTTL         = time.Hour * 3
	TrailStopPoints = 15
	ShortDt         = "02/01 15:04:05"
	ShortTime       = "15:04:05"
)

type MarketConfig interface {
	MarketOpen(t time.Time) time.Time
	MarketClose(t time.Time) time.Time
	IsOpen(t time.Time) bool
}

type EntryScanner interface {
	On5MinBar(history *History, tradeManager *TradeManager)
}

type exchange interface {
	CreateOrder(direction Direction, open, stop, target float64) *ExTrade
	ExitPosition(id int) *ExTrade
	CancelOrder(id int)
}

type TradeManager struct {
	aggregator    *BarAggregator
	exchange      exchange
	entryScanners []EntryScanner
	orders        map[int]*Trade
	positions     map[int]*Trade
	activeSignals *Signals
	marketConfig  MarketConfig
	history       *History
}

func (t *TradeManager) PositionClosed(exTrade *ExTrade) {
	trade := t.positions[exTrade.Id]
	trade.updateClosed(exTrade)
	tracked := t.positions[trade.Id]
	if tracked == nil {
		log.Printf("position not found: %+v", trade)
		return
	}
	delete(t.positions, trade.Id)
}

func (t *TradeManager) PositionOpened(exTrade *ExTrade) {
	trade := t.orders[exTrade.Id]
	if trade == nil {
		log.Printf("order not found: %+v", exTrade)
		t.exchange.ExitPosition(exTrade.Id)
		return
	}
	trade.Signal.Trades = append(trade.Signal.Trades, trade)
	t.positions[trade.Id] = trade
	for id := range t.orders {
		if id != trade.Id {
			t.exchange.CancelOrder(id)
		}
	}
	t.orders = make(map[int]*Trade)
}

func (t *TradeManager) HandleTick(tick *Tick) {
	bar := t.aggregator.ProcessTick(tick)
	if bar != nil {
		t.history.AddBar(bar)

		if !t.marketConfig.IsOpen(bar.Timestamp) {
			// don't go over closing time
			if len(t.orders) > 0 {
				for id := range t.orders {
					t.exchange.CancelOrder(id)
				}
				t.orders = make(map[int]*Trade)
			}
			if len(t.positions) > 0 {
				for id, trade := range t.positions {
					trade.ExTrade = t.exchange.ExitPosition(id)
					trade.ExitReason = ExitReasonMarketClose
				}
				t.positions = make(map[int]*Trade)
			}
		} else {
			for _, scanner := range t.entryScanners {
				scanner.On5MinBar(t.history, t)
			}
		}
	}
}

func (t *TradeManager) SetExchange(exchange exchange) {
	t.exchange = exchange
}

func (t *TradeManager) AddSignal(signal *Signal) {
	t.history.AddSignal(signal)
}

func (t *TradeManager) CreateOrder(signal *Signal, reason OpenReason, direction Direction, open, stop, target float64) *Trade {
	exTrade := t.exchange.CreateOrder(direction, open, stop, target)
	trade := &Trade{
		ExTrade:    exTrade,
		OpenReason: reason,
	}
	t.orders[trade.Id] = trade
	trade.Signal = signal
	return trade
}

func (t *TradeManager) PrintReport() {
	t.history.PrintReport()
}

func (t *TradeManager) SaveData(dir string) {
	t.history.SaveData(dir)
}

func NewTradeManager(marketConfig MarketConfig, scanners ...EntryScanner) *TradeManager {
	rv := &TradeManager{
		aggregator:    NewBarAggregator(time.Minute * 5),
		entryScanners: scanners,
		marketConfig:  marketConfig,
		orders:        make(map[int]*Trade),
		positions:     make(map[int]*Trade),
		history:       NewHistory(),
	}
	return rv
}
