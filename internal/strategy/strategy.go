package strategy

import (
	"log"
	"math"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
)

//const (
//	AddToPositionPoints         = 20
//	AddToTradeVelocityThreshold = 10
//)

type MarketContext struct {
	exchange           models.Exchange
	aggregator         *models.BarAggregator
	lastTick           *models.Tick
	orders             map[int]*models.Trade
	positions          map[int]*models.Trade
	history            *models.History
	CancelOrdersOnFill bool
	now                time.Time
	onNewBarCb         func()
	onNewTickCb        func(*models.Tick)
}

func (t *MarketContext) calculateTradeSize(stopPoints float64) float64 {
	risk := 0.01 * t.exchange.GetBalance() // 1% of account size
	tradeSize := risk / stopPoints
	tradeSize = math.Max(tradeSize, 1) // ensure minimum size is 1
	return internal.Round2(tradeSize)
}

func (t *MarketContext) PositionClosed(exTrade *models.Position) {
	trade := t.positions[exTrade.Id]
	trade.UpdateClosed(exTrade)
	tracked := t.positions[trade.Id]
	if tracked == nil {
		log.Printf("position not found: %+v", trade)
		return
	}
	delete(t.positions, trade.Id)
}

func (t *MarketContext) PositionOpened(exTrade *models.Position) {
	trade := t.orders[exTrade.Id]
	if trade == nil {
		log.Printf("order not found: %+v", exTrade)
		t.ExitPosition(exTrade.Id)
		return
	}
	trade.StopLog = append(trade.StopLog, &models.StopLog{
		Stop:      trade.StopPrice,
		Timestamp: trade.EntryTime,
	})
	trade.Signal.AddPosition(trade)
	t.positions[trade.Id] = trade

	if t.CancelOrdersOnFill {
		for id := range t.orders {
			if id != trade.Id {
				t.exchange.CancelOrder(id)
			}
		}
		t.orders = make(map[int]*models.Trade)
	}
}

func (t *MarketContext) AggregateTick(tick *models.Tick) *models.Bar {
	t.lastTick = tick
	bar := t.aggregator.AddTick(tick)

	if tick != nil {
		t.now = tick.Timestamp // simulate time
	}

	if bar != nil {
		t.history.AddBar(bar)
	}
	return bar
}

func (t *MarketContext) SetExchange(exchange models.Exchange) {
	t.exchange = exchange
}

func (t *MarketContext) AddSignal(signal *models.Signal) {
	t.history.AddSignal(signal)
}

func (t *MarketContext) CreateOrder(symbol models.Symbol, signal *models.Signal, reason models.OpenReason, direction models.Direction, open, stop, target float64) *models.Trade {
	stopPts := internal.Round2(func() float64 {
		if direction == models.Long {
			return open - stop
		}
		return stop - open
	}())
	size := t.calculateTradeSize(stopPts)
	exTrade := t.exchange.CreateOrder(symbol, direction, size, open, stop, target)
	trade := &models.Trade{
		Position:        exTrade,
		OpenReason:      reason,
		TrailStopPoints: stopPts,
	}
	t.orders[trade.Id] = trade
	trade.Signal = signal
	return trade
}

func (t *MarketContext) PrintReport() {
	t.history.PrintReport()
}

func (t *MarketContext) UpdatePosition(trade *models.Trade, stop, target float64) {
	if len(trade.StopLog) > 0 {
		lastStop := trade.StopLog[len(trade.StopLog)-1]
		lastBar := t.history.GetBar(0)
		// only update stop if we are in a newer bar than the last stop
		if lastBar.EndTime().Equal(lastStop.Timestamp) || lastBar.EndTime().Before(lastStop.Timestamp) {
			return
		}
	}

	trade.StopLog = append(trade.StopLog, &models.StopLog{
		Stop:      stop,
		Timestamp: t.Now(),
	})
	t.exchange.UpdatePosition(trade.Id, stop, target)
}

func (t *MarketContext) Backfill(bars models.Series) {
	for _, bar := range bars {
		t.history.AddBar(bar)
	}
}

func (t *MarketContext) SaveData(dir string, location *time.Location) {
	t.history.SaveData(dir, location)
}

func (t *MarketContext) Now() time.Time {
	// TODO: if !prod { ...
	return t.now
}

func (t *MarketContext) CloseAll(reason models.ExitReason) {
	t.CloseAllOrders()
	t.CloseAllPositions(reason)
}

func (t *MarketContext) ExitPosition(id int) *models.Position {
	return t.exchange.ExitPosition(id, t.lastTick)
}

func (t *MarketContext) CloseAllOrders() {
	if len(t.orders) > 0 {
		for id := range t.orders {
			t.exchange.CancelOrder(id)
		}
		t.orders = make(map[int]*models.Trade)
	}
}

func (t *MarketContext) CloseAllPositions(reason models.ExitReason) {
	if len(t.positions) > 0 {
		for id, trade := range t.positions {
			trade.Position = t.ExitPosition(id)
			trade.ExitReason = reason
		}
		t.positions = make(map[int]*models.Trade)
	}
}

func NewMarketContext(barDuration time.Duration) *MarketContext {
	rv := &MarketContext{
		aggregator: models.NewBarAggregator(barDuration),
		orders:     make(map[int]*models.Trade),
		positions:  make(map[int]*models.Trade),
		history:    models.NewHistory(),
	}
	return rv
}
