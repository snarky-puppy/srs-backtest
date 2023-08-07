package exchange

import (
	"log"
	"math"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
)

const (
	AddToPositionPoints         = 20
	ShortDt                     = "02/01 15:04:05"
	ShortTime                   = "15:04:05"
	AddToTradeVelocityThreshold = 10
	AngleRecordBars             = 10
)

type MarketConfig interface {
	MarketOpen(t time.Time) time.Time
	MarketClose(t time.Time) time.Time
	IsOpen(t time.Time) bool
	Location() *time.Location
}

type Platform interface {
	CreateOrder(symbol models.Symbol, direction models.Direction, size, open, stop, target float64) *models.Trade
	ExitPosition(id int) *models.Trade
	CancelOrder(id int)
	UpdatePosition(id int, stop, target float64)
	GetBalance() float64
}

type MarketContext struct {
	exchange           Platform
	aggregator         *BarAggregator
	scanner            EntryScanner
	orders             map[int]*Trade
	positions          map[int]*Trade
	activeSignals      *Signals
	marketConfig       MarketConfig
	history            *History
	CancelOrdersOnFill bool
}

func (t *MarketContext) calculateTradeSize(stopPoints float64) float64 {
	risk := 0.01 * t.exchange.GetBalance() // 1% of account size
	tradeSize := risk / stopPoints
	tradeSize = math.Max(tradeSize, 1) // ensure minimum size is 1
	return internal.Round2(tradeSize)
}

func (t *MarketContext) PositionClosed(exTrade *models.Trade) {
	trade := t.positions[exTrade.Id]
	trade.updateClosed(exTrade)
	tracked := t.positions[trade.Id]
	if tracked == nil {
		log.Printf("position not found: %+v", trade)
		return
	}
	delete(t.positions, trade.Id)
}

func (t *MarketContext) PositionOpened(exTrade *models.Trade) {
	trade := t.orders[exTrade.Id]
	if trade == nil {
		log.Printf("order not found: %+v", exTrade)
		t.exchange.ExitPosition(exTrade.Id)
		return
	}
	trade.StopLog = append(trade.StopLog, &StopLog{
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
		t.orders = make(map[int]*Trade)
	}
}

func (t *MarketContext) HandleTick(tick *models.Tick) {
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
					trade.Trade = t.exchange.ExitPosition(id)
					trade.ExitReason = ExitReasonMarketClose
				}
				t.positions = make(map[int]*Trade)
			}
		} else {
			t.On5MinBar(tick, bar)
			t.scanner.On5MinBar(t.history, t)
		}
	}
	t.managePositions(tick)
}

func (t *MarketContext) SetExchange(exchange Platform) {
	t.exchange = exchange
}

func (t *MarketContext) AddSignal(signal *Signal) {
	t.history.AddSignal(signal)
}

func (t *MarketContext) CreateOrder(symbol models.Symbol, signal *Signal, reason OpenReason, direction models.Direction, open, stop, target float64) *Trade {
	stopPts := internal.Round2(func() float64 {
		if direction == models.Long {
			return open - stop
		}
		return stop - open
	}())
	stopPts = math.Min(stopPts, MaxStopPts)
	size := t.calculateTradeSize(stopPts)
	exTrade := t.exchange.CreateOrder(symbol, direction, size, open, stop, target)
	trade := &Trade{
		Trade:            exTrade,
		OpenReason:       reason,
		AutoAdjustStop:   true,
		LoserThreshold:   15,
		CanAddToPosition: true,
		TrailStopPoints:  stopPts,
	}
	t.orders[trade.Id] = trade
	trade.Signal = signal
	return trade
}

func (t *MarketContext) PrintReport() {
	t.history.PrintReport()
}

func (t *MarketContext) SaveData(dir string) {
	t.history.SaveData(dir, t.marketConfig.Location())
}

func (t *MarketContext) managePositions(tick *models.Tick) {
	for _, trade := range t.positions {
		t.managePosition(tick, trade)
	}
}

func (t *MarketContext) considerAddingToPosition(bar *Bar, winner *Trade) {

	if winner.EntryTime.Truncate(bar.Duration).Equal(bar.Timestamp) {
		return
	}

	if calculatePointsProfit(winner.Trade, bar.AvgPrice()) < AddToPositionPoints {
		return
	}

	// crude velocity check
	switch winner.Direction {
	case models.Long:
		if t.history.Sma5.Calculate()-t.history.Sma25.Calculate() < AddToTradeVelocityThreshold {
			return
		}
	case models.Short:
		if t.history.Sma25.Calculate()-t.history.Sma5.Calculate() < AddToTradeVelocityThreshold {
			return
		}
	}

	winner.CanAddToPosition = false // a trade can only be added to once

	// if we are long, add an order at the bottom of the previous bar low
	// if we are short, add an order at the top of the previous bar high

	open := bar.Open
	switch winner.Direction {
	case models.Long:
		open = bar.Low
	case models.Short:
		open = bar.High
	}

	newTrade := t.CreateOrder(winner.Symbol, winner.Signal, OpenReasonAddToPosition, winner.Direction, open, winner.StopPrice, 0)
	newTrade.IsAdditional = true
	//newTrade.BTL = 5
}

func (t *MarketContext) managePosition(tick *models.Tick, trade *Trade) {

	tickPrice := tick.MidPrice()

	if trade.AutoAdjustStop {
		switch trade.Direction {
		case models.Long:
			// trail by 30 pts
			if (tickPrice - trade.TrailStopPoints) > trade.StopPrice {
				t.UpdatePosition(tick, trade, tickPrice-trade.TrailStopPoints, 0)
			}
		case models.Short:
			if (tickPrice + trade.TrailStopPoints) < trade.StopPrice {
				t.UpdatePosition(tick, trade, tickPrice+trade.TrailStopPoints, 0)
			}
		}
	}
}

func (t *MarketContext) UpdatePosition(tick *models.Tick, trade *Trade, stop, target float64) {
	if len(trade.StopLog) > 0 {
		lastStop := trade.StopLog[len(trade.StopLog)-1]
		lastBar := t.history.GetBar(0)
		// only update stop if we are in a newer bar than the last stop
		if lastBar.EndTime().Equal(lastStop.Timestamp) || lastBar.EndTime().Before(lastStop.Timestamp) {
			return
		}
	}

	trade.StopLog = append(trade.StopLog, &StopLog{
		Stop:      stop,
		Timestamp: tick.Timestamp,
	})
	t.exchange.UpdatePosition(trade.Id, stop, 0)
}

func (t *MarketContext) On5MinBar(tick *models.Tick, bar *Bar) {

	for _, position := range t.positions {
		position.CheckLoser(bar)
		if position.IsLoser() {
			// try to close for beak-even
			switch position.Direction {
			case models.Long:
				t.exchange.UpdatePosition(position.Id, math.Max(position.StopPrice, bar.Low+3), position.OpenPrice)
			case models.Short:
				t.exchange.UpdatePosition(position.Id, math.Min(position.StopPrice, bar.High+3), position.OpenPrice)
			}
			continue
		}

		if position.Id == 13 {
			log.Println("here")
		}

		// if we are long, and the 25 SMA crosses below the 5 SMA, exit the position
		if position.Direction == models.Long {
			if t.history.Sma25.CrossedOver(t.history.Sma5, 2) {
				if calculatePointsProfit(position.Trade, bar.Close) <= 0 {
					// if not in profit, tighten the stop and try to close for break even
					stop := t.history.FindAverageLow(5)
					position.ExitReason = ExitReasonSmaCrossStop
					t.exchange.UpdatePosition(position.Id, stop, position.EntryPrice)
				} else {
					// otherwise, just close for profit
					position.ExitReason = ExitReasonSmaCross
					t.exchange.ExitPosition(position.Id)
				}
				continue
			}
		} else {
			if t.history.Sma5.CrossedOver(t.history.Sma25, 2) {
				if calculatePointsProfit(position.Trade, bar.Close) <= 0 {
					// if not in profit, tighten the stop and try to close for break even
					stop := t.history.FindAverageHigh(5)
					position.ExitReason = ExitReasonSmaCrossStop
					t.exchange.UpdatePosition(position.Id, stop, position.EntryPrice)
				} else {
					// otherwise, just close for profit
					position.ExitReason = ExitReasonSmaCross
					t.exchange.ExitPosition(position.Id)
				}
				continue
			}
		}

		if position.CanAddToPosition {
			t.considerAddingToPosition(bar, position)
		}
	}
}

func NewMarketContext(scanner EntryScanner) *MarketContext {
	rv := &MarketContext{
		aggregator: NewBarAggregator(time.Minute * 5),
		scanner:    scanner,
		orders:     make(map[int]*Trade),
		positions:  make(map[int]*Trade),
		history:    NewHistory(),
	}
	return rv
}
