package strategy

import (
	"log"
	"time"

	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/models"
)

const (
	DATA     = "Germany 40 - Rolling Cash"
	MarketID = 17068
	QuoteID  = 6374
)

func (s *SrsEntry) Location() *time.Location {
	return s.Timezone()
}

func (s *SrsEntry) IsOpen(t time.Time) bool {
	t = t.In(s.Timezone())
	return t.After(s.MarketOpen(t)) && t.Before(s.MarketClose(t))
}

func (s *SrsEntry) MarketOpen(t time.Time) time.Time {
	t = t.In(s.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, t.Location())
}

// MarketClose returns the time the market closes on day t
func (s *SrsEntry) MarketClose(t time.Time) time.Time {
	t = t.In(s.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

func (s *SrsEntry) Timezone() *time.Location {
	if s.tz == nil {
		var err error
		s.tz, err = time.LoadLocation("Europe/Berlin")
		if err != nil {
			panic(err)
		}
	}
	return s.tz
}

func (s *SrsEntry) isPeriod(t time.Time) bool {
	t = t.In(s.Timezone())
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}

func (s *SrsEntry) On5MinBar(history *exchange.History, marketContext *exchange.MarketContext) {
	bar := history.GetBar(0)

	if bar == nil || bar.Timestamp.Weekday() == time.Tuesday {
		return
	}

	t := bar.Timestamp.In(s.Timezone())

	// setup srs signal
	if s.isPeriod(t) {

		// take last 3 elements from s.bars
		const historySize = 3
		setupBars := history.GetBars(historySize)
		if len(setupBars) != historySize {
			panic("expected n bars")
		}
		if setupBars[0].Timestamp.Before(s.MarketOpen(t)) {
			log.Println("skipping signal due to gap in data", setupBars[0].Timestamp, setupBars[1].Timestamp, setupBars[2].Timestamp)
			return
		}
		signalBar := setupBars[0].Copy()
		for _, b := range setupBars[1:] {
			signalBar.Add(b)
		}
		signalBar.Duration = historySize * 5 * time.Minute
		s.signal = &exchange.Signal{
			Bar: signalBar,
			CanTradeFn: func(signal *exchange.Signal, direction models.Direction) bool {
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
			TryMaxStop:    true,
			EnableSmaExit: false,
		}
		marketContext.AddSignal(s.signal)

		createOrder := func(direction models.Direction) {
			entry, stop, _ := s.signal.EST(direction)
			marketContext.CreateOrder(
				s.Symbol(),
				s.signal,
				exchange.OpenReasonSignal,
				direction,
				entry,
				stop,
				0)
		}

		// add 2 trades
		createOrder(models.Long)
		createOrder(models.Short)
	}

	if s.signal == nil {
		return
	}

}

func (s *SrsEntry) SimData() string {
	return DATA
}

func (s *SrsEntry) Symbol() models.Symbol {
	return models.Symbol{
		MarketID: MarketID,
		QuoteID:  QuoteID,
	}
}

type SrsEntry struct {
	active bool
	signal *exchange.Signal
	tz     *time.Location
}
