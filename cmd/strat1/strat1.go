package main

import (
	"log"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/pp"
)

const DATA = "data/td/Germany 40 - Rolling Cash.csv.gz"
const (
	ScanTTL         = time.Hour * 3
	TrailStopPoints = 15
)

func main() {
	ctx := internal.Graceful()
	strat1 := Strategy1{
		history: pp.NewHistory(),
	}
	exch := exchange.NewExchange(ctx, DATA, &strat1)
	exch.Wait()
	log.Println("read", len(strat1.bars), "bars")
}

// MarketClose returns the time the market closes on day t
func MarketClose(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

func MarketOpen(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, t.Location())
}

type Strategy1 struct {
	srsSignal   *pp.Signal
	rangeSignal *pp.Signal
	trades      []*pp.Trade
	history     *pp.History
	exchange    *exchange.Exchange
}

func (s *Strategy1) SetExchange(exch *exchange.Exchange) {
	s.exchange = exch
}

func (s *Strategy1) TradeOpened(trade *pp.Trade) {
	//TODO implement me
	panic("implement me")
}

func (s *Strategy1) TradeStoppedOut(trade *pp.Trade) {
}

func (s *Strategy1) FiveMinBar(bar *pp.Bar) {
	s.history.AddBar(bar)
	t := bar.Timestamp
	if t.Before(MarketOpen(t)) || t.After(MarketClose(t)) {
		// don't go over closing time
		s.closeAll("market close", bar)
		s.srsSignal = nil
		s.rangeSignal = nil
		return
	}

	if len(s.trades) > 0 {
		s.manageTrades()
	} else {
		s.scanForSignal()
	}
}

func (s *Strategy1) closeAll(reason string, bar *pp.Bar) {

}

func (s *Strategy1) manageTrades() {

}

// IsPeriod returns true if the given time is the end of the 15 minute interval we are using for signals
func (s *Strategy1) isPeriod(t time.Time) bool {
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}

func (s *Strategy1) scanForSignal() {
	bar := s.history.GetBar(0)
	prevBar := s.history.GetBar(-1)
	t := bar.Timestamp

	// setup srs signal
	if s.isPeriod(t) {

		// take last 3 elements from s.bars
		last3Bars := s.history.GetBars(-3)
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
		s.srsSignal = &pp.Signal{
			Bar:         signalBar,
			Idx:         i - 3,
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
		s.history.AddSignal(s.srsSignal)

		// add 2 trades
		longTrade := s.srsSignal.NewTrade(pp.Long)
		shortTrade := s.srsSignal.NewTrade(pp.Short)
		s.exchange.Trades.CreateTrade(longTrade)
		s.exchange.Trades.CreateTrade(shortTrade)
	}

	// check srs signal crossed
	if s.srsSignal != nil {

		// if our signal is good, enter the trade
		// price has to "cross over" the signal line
		high := s.srsSignal.High()
		low := s.srsSignal.Low()
		if prevBar.High <= high && bar.High > high && s.srsSignal.CanTrade(pp.Long) {
			entry := high // stop order was set here
			stopPrice := stop(s.srsSignal.Bar, pp.Long)
			trade := pp.NewTrade(pp.Long, entry, stop(s.srsSignal.Bar, pp.Long), target(entry, pp.Long), 0, bar, s.srsSignal, "srs crossover")
			s.srsSignal.Trades = append(s.srsSignal.Trades, trade)
			s.trades = append(s.trades, trade)
			return

		} else if prevBar.Low >= low && bar.Low < low && s.srsSignal.CanTrade(pp.Short) {
			entry := low // stop order was set here
			stopPrice := stop(s.srsSignal.Bar, pp.Short)
			trade := pp.NewTrade(pp.Short, entry, stopPrice, target(entry, pp.Short), i, bar, s.srsSignal, "srs crossover")
			s.srsSignal.Trades = append(s.srsSignal.Trades, trade)
			s.trades = append(s.trades, trade)
			return
		}
	}

	// setup range signal
	if s.rangeSignal == nil {
		// detect trading range in last 10 bars
		idx := i - 10
		tRange := bars[idx].Copy()
		if tRange.Timestamp.Before(MarketOpen(t)) {
			continue
		}

		for _, b := range bars[idx+1 : i] {
			tRange.Add(b)
		}

		if tRange.High-tRange.Low <= 20 {
			rangeSignal = &pp.Signal{
				Bar:         tRange,
				Idx:         idx,
				BarDuration: 10 * (5 * time.Minute),
				CanTradeFn: func(s pp.Signal, direction pp.Direction) bool {
					return len(s.Trades) == 0
				},
			}
			s.Signals = append(s.Signals, rangeSignal)
		}
	}

	// check range signal crossed
	if s.rangeSignal != nil {
		// if our signal is good, enter the trade
		// price has to "cross over" the signal line
		high := rangeSignal.High()
		low := rangeSignal.Low()
		if prevBar.High <= high && bar.High > high && rangeSignal.CanTrade(pp.Long) {
			entry := high // stop order was set here
			stopPrice := stop(rangeSignal.Bar, pp.Long)
			trade := pp.NewTrade(pp.Long, entry, stopPrice, target(entry, pp.Long), i, bar, rangeSignal, "range crossover")
			rangeSignal.Trades = append(rangeSignal.Trades, trade)
			rangeSignal = nil // cannot use again
			trades = append(trades, trade)
			continue

		} else if prevBar.Low >= low && bar.Low < low && rangeSignal.CanTrade(pp.Short) {
			entry := low // stop order was set here
			stopPrice := stop(rangeSignal.Bar, pp.Short)
			trade := pp.NewTrade(pp.Short, entry, stopPrice, target(entry, pp.Short), i, bar, rangeSignal, "range crossover")
			rangeSignal.Trades = append(rangeSignal.Trades, trade)
			rangeSignal = nil // cannot use again
			trades = append(trades, trade)
			continue
		}
	}
}
