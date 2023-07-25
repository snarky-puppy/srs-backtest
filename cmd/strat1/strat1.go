package main

import (
	"log"
	"time"

	"github.com/mwlazlo/srs/internal/exchange"
)

const DATA = "data/td/Germany 40 - Rolling Cash.csv.gz"

type DaxConfig struct {
	tz *time.Location
}

func (d *DaxConfig) IsOpen(t time.Time) bool {
	t = t.In(d.Timezone())
	return t.After(d.MarketOpen(t)) && t.Before(d.MarketClose(t))
}

func (d *DaxConfig) MarketOpen(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, t.Location())
}

// MarketClose returns the time the market closes on day t
func (d *DaxConfig) MarketClose(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

func (d *DaxConfig) Timezone() *time.Location {
	if d.tz == nil {
		var err error
		d.tz, err = time.LoadLocation("Europe/Berlin")
		if err != nil {
			panic(err)
		}
	}
	return d.tz
}

func (d *DaxConfig) isPeriod(t time.Time) bool {
	t = t.In(d.Timezone())
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}

var daxConfig = &DaxConfig{}

// IsPeriod returns true if the given time is the end of the 15-minute interval we are using for signals

type SrsEntry struct {
	active bool
	signal *exchange.Signal
}

func (s *SrsEntry) On5MinBar(history *exchange.History, tradeManager *exchange.TradeManager) {
	bar := history.GetBar(0)

	if bar == nil {
		return
	}

	t := bar.Timestamp.In(daxConfig.Timezone())

	// setup srs signal
	if daxConfig.isPeriod(t) {

		// take last 3 elements from s.bars
		last3Bars := history.GetBars(3)
		if len(last3Bars) != 3 {
			panic("expected 3 bars")
		}
		if last3Bars[0].Timestamp.Before(daxConfig.MarketOpen(t)) {
			log.Println("skipping signal due to gap in data", last3Bars[0].Timestamp, last3Bars[1].Timestamp, last3Bars[2].Timestamp)
			return
		}
		signalBar := last3Bars[0].Copy()
		for _, b := range last3Bars[1:] {
			signalBar.Add(b)
		}
		signalBar.Duration = 15 * time.Minute
		s.signal = &exchange.Signal{
			Bar: signalBar,
			CanTradeFn: func(signal *exchange.Signal, direction exchange.Direction) bool {
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
		tradeManager.AddSignal(s.signal)

		createOrder := func(direction exchange.Direction) {
			tradeManager.CreateOrder(
				s.signal,
				exchange.OpenReasonSignal,
				direction,
				s.signal.Entry(direction),
				s.signal.Stop(direction),
				s.signal.Target(direction))
		}

		// add 2 trades
		createOrder(exchange.Long)
		createOrder(exchange.Short)
	}

	if s.signal == nil {
		return
	}

}

func main() {
	tradeManager := exchange.NewTradeManager(daxConfig, &SrsEntry{})
	sim := exchange.NewExchangeSimulator(DATA, tradeManager)
	tradeManager.SetExchange(sim)

	sim.ProcessTicks()
	tradeManager.PrintReport()
	tradeManager.SaveData("data/reports")
}
