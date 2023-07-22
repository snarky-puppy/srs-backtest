package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/pp"
)

const DATA = "data/td/Germany 40 - Rolling Cash.csv.gz"
const (
	ScanTTL         = time.Hour * 3
	TrailStopPoints = 15
	ShortDt         = "02/01 15:04:05"
	ShortTime       = "15:04:05"
)

func main() {
	tradeManager := exchange.NewTradeManager(SrsEntry)
	sim := exchange.NewExchangeSimulator(DATA, tradeManager)
	tradeManager.SetExchange(sim)

	sim.ProcessTicks()
	tradeManager.PrintReport()
	tradeManager.SaveData("data/reports")
}

func SrsEntry(history *pp.History, tradeManager *exchange.TradeManager) {
	bar := history.GetBar(0)

	if bar == nil {
		return
	}

	t := bar.Timestamp

	// setup srs signal
	if isPeriod(t) {

		// take last 3 elements from s.bars
		last3Bars := history.GetBars(3)
		if len(last3Bars) != 3 {
			panic("expected 3 bars")
		}
		if last3Bars[0].Timestamp.Before(MarketOpen(t)) {
			log.Println("skipping signal due to gap in data", last3Bars[0].Timestamp, last3Bars[1].Timestamp, last3Bars[2].Timestamp)
			return
		}
		signalBar := last3Bars[0].Copy()
		for _, b := range last3Bars[1:] {
			signalBar.Add(b)
		}
		signalBar.Duration = 15 * time.Minute
		rv := &pp.Signal{
			Bar:         signalBar,
			BarDuration: 15 * time.Minute,
			CanTradeFn: func(signal pp.Signal, direction pp.Direction) bool {
				// we can trade on each direction only twice
				cnt := 0
				for _, trade := range signal.Trades {
					if trade.Direction == direction && !trade.IsAdditional {
						cnt++
						if cnt >= 3 {
							return false
						}
					}
				}
				return true
			},
		}
		tradeManager.AddSignal(rv)

		// add 2 trades
		longTrade := rv.NewTrade(pp.Long)
		shortTrade := rv.NewTrade(pp.Short)
		tradeManager.CreateOrder(longTrade)
		tradeManager.CreateOrder(shortTrade)
	}
}

// MarketClose returns the time the market closes on day t
func MarketClose(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

func MarketOpen(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, t.Location())
}

type Strategy1 struct {
	srsSignal *pp.Signal
	//rangeSignal *pp.Signal
	history  *pp.History
	exchange *exchange.TradeManager
}

func (s *Strategy1) SetExchange(exch *exchange.TradeManager) {
	s.exchange = exch
}

func (s *Strategy1) PositionOpened(trade *pp.Trade) {
	s.trades = append(s.trades, trade)
	s.exchange.Trades.CloseAllOrders()
	s.srsSignal.AddPosition(trade)
}

func (s *Strategy1) OrderOpened(trade *pp.Trade) {
	s.orders = append(s.orders, trade)
}

func (s *Strategy1) TradeClosed(trade *pp.Trade, reason exchange.TradeCloseReason) {
	fmt.Println("trade", trade.OpenTime.Format(ShortDt), trade.CloseTime.Format(ShortTime), trade.Direction, reason, trade.Profit())
	trades := []*pp.Trade{}
	for _, t := range s.trades {
		if t.Id != trade.Id {
			trades = append(trades, t)
		}
	}
	s.trades = trades
}

func (s *Strategy1) FiveMinBar(bar *pp.Bar) {
	s.history.AddBar(bar)

	t := bar.Timestamp

	if t.Before(MarketOpen(t)) || t.After(MarketClose(t)) {
		// don't go over closing time
		s.exchange.Trades.CloseAllPositions(bar)
		s.srsSignal = nil
		//s.rangeSignal = nil
		return
	}

	if len(s.trades) > 0 {
		s.manageTrades()
	} else {
		s.scanForSignal()
	}
}

// IsPeriod returns true if the given time is the end of the 15 minute interval we are using for signals
func isPeriod(t time.Time) bool {
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}
