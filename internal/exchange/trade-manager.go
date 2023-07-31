package exchange

import (
	"log"
	"math"
	"time"
)

const (
	ScanTTL             = time.Hour * 3
	TrailStopPoints     = 30
	AddToPositionPoints = 20
	ShortDt             = "02/01 15:04:05"
	ShortTime           = "15:04:05"
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
	UpdatePosition(id int, stop, target float64)
}

type TradeManager struct {
	aggregator         *BarAggregator
	exchange           exchange
	entryScanners      []EntryScanner
	orders             map[int]*Trade
	positions          map[int]*Trade
	activeSignals      *Signals
	marketConfig       MarketConfig
	history            *History
	CancelOrdersOnFill bool
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
	trade.StopLog = append(trade.StopLog, &StopLog{
		Stop:      trade.StopPrice,
		Timestamp: trade.EntryTime,
	})
	trade.Signal.Trades = append(trade.Signal.Trades, trade)
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
			t.On5MinBar(tick, bar)
			for _, scanner := range t.entryScanners {
				scanner.On5MinBar(t.history, t)
			}
		}
	}
	t.managePositions(tick)
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
		ExTrade:          exTrade,
		OpenReason:       reason,
		AutoAdjustStop:   true,
		LoserThreshold:   15,
		CanAddToPosition: true,
		TrailStopPoints: func() float64 {
			if direction == Long {
				return open - stop
			}
			return stop - open
		}(),
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

func (t *TradeManager) managePositions(tick *Tick) {
	for _, trade := range t.positions {
		t.managePosition(tick, trade)
	}
}

func (t *TradeManager) considerAddingToPosition(bar *Bar, winner *Trade) *Trade {
	winner.CanAddToPosition = false // only 1 chance to add to position

	bars := t.history.GetBars(-5)

	// before: total profits: 37601
	// after:  total profits: 19967
	//if winner.Loser >= 2.5 {
	//	return nil
	//}

	// we would add an entry at the top of the previous bar high
	// so if this bar doesn't reach that high, we don't add
	// before: 43092
	// after: 55090
	switch winner.Direction {
	case Long:
		if bar.High < bars.GetBar(-1).High {
			return nil
		}
	case Short:
		if bar.Low > bars.GetBar(-1).Low {
			return nil
		}
	}

	reachedTarget := winner.TargetPrice
	half := 15.0 // float64(TargetPoints / 2)
	// adjust stop to target / 2
	switch winner.Direction {
	case Long:
		t.exchange.UpdatePosition(winner.Id, winner.TargetPrice-half, 0)
	case Short:
		t.exchange.UpdatePosition(winner.Id, winner.TargetPrice+half, 0)
	}
	winner.AutoAdjustStop = true

	newTrade := t.CreateOrder(winner.Signal, OpenReasonAddToPosition, winner.Direction, reachedTarget, winner.StopPrice, winner.TargetPrice)
	newTrade.AutoAdjustStop = true
	newTrade.DisableLoserCheck = true
	newTrade.IsAdditional = true
	return newTrade
}

func (t *TradeManager) managePosition(tick *Tick, trade *Trade) {

	tickPrice := tick.MidPrice()

	// region adjust stops
	// w/o 80,50
	// w 75,45

	if trade.AutoAdjustStop {
		switch trade.Direction {
		case Long:
			// trail by 30 pts
			if (tickPrice - trade.TrailStopPoints) > trade.StopPrice {
				t.UpdatePosition(tick, trade, tickPrice-trade.TrailStopPoints, 0)
			}
		case Short:
			if (tickPrice + trade.TrailStopPoints) < trade.StopPrice {
				t.UpdatePosition(tick, trade, tickPrice+trade.TrailStopPoints, 0)
			}
		}
	}
	// endregion

	/*

			if a trade triggers and does well (a screamer), but then trends back to the trigger,
			don't take the next trade if it triggers (trending wrong way)
			2000-06-07 09:10 http://localhost:8081/?signalIndex=5

			trade triggers, make a high, dips and makes a double top/bottom,
			close at the peak (and enter the opposite direction?)
			2000-06-12 09:10 http://localhost:8081/?signalIndex=8

			don't have more than 1 sell/buy in a row
		    BUT sometimes this works, test it, remove the restriction
			and count how many times 2 buys in a row profit, or how many times the 2nd fails, etc
			2000-06-13 09:10 http://localhost:8081/?signalIndex=9

			often, after reaching the target, the price will plateau for a few bars
			perhaps create new signal area (box shape) on that plateau
			2000-06-06 09:10 http://localhost:8081/?signalIndex=4

			price is trending -- add to trade
			how to know if it's a trend or screamer?

	*/

	/*
		// region target
		switch trade.Direction {
		case pp.Long:
			if bar.High >= trade.Target {
				trade.CanAddToPosition = true
				return trade
			}
		case pp.Short:
			if bar.Low <= trade.Target {
				trade.CanAddToPosition = true
				return trade
			}
		}
		// endregion target

		return trade

	*/
}

func (t *TradeManager) UpdatePosition(tick *Tick, trade *Trade, stop, target float64) {
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

func (t *TradeManager) On5MinBar(tick *Tick, bar *Bar) {
	for _, position := range t.positions {
		position.CheckLoser(bar)
		if position.IsLoser() {
			// try to close for beak-even
			switch position.Direction {
			case Long:
				t.exchange.UpdatePosition(position.Id, math.Max(position.StopPrice, bar.Low+3), position.OpenPrice)
			case Short:
				t.exchange.UpdatePosition(position.Id, math.Min(position.StopPrice, bar.High+3), position.OpenPrice)
			}
			continue
		}

		if !position.IsAdditional && position.CanAddToPosition {
			t.considerAddingToPosition(bar, position)
		}
	}
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
