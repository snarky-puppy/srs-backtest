package strategy

import (
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

func (r *Test) Location() *time.Location {
	return r.Timezone()
}

func (r *Test) IsOpen(tm time.Time) bool {
	tm = tm.In(r.Timezone())
	return tm.After(r.MarketOpen(tm)) && tm.Before(r.MarketClose(tm))
}

func (r *Test) MarketOpen(tm time.Time) time.Time {
	tm = tm.In(r.Timezone())
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 9, 0, 0, 0, r.Location())
}

// MarketClose returns the time the market closes on day t
func (r *Test) MarketClose(tm time.Time) time.Time {
	tm = tm.In(r.Timezone())
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 17, 25, 0, 0, r.Location())
}

func (r *Test) Timezone() *time.Location {
	if r.tz == nil {
		var err error
		r.tz, err = time.LoadLocation("America/New_York")
		if err != nil {
			panic(err)
		}
	}
	return r.tz
}

func (r *Test) isPeriod(tm time.Time) bool {
	tm = tm.In(r.Timezone())
	if tm.Minute() == 25 && tm.Hour() == 9 {
		return true
	}
	return false
}

func (r *Test) On1MinBar() {
	bar := r.history.GetBar(0)

	if bar == nil {
		return
	}

	r.MarketContext.CloseAll("")

	if !r.IsOpen(bar.Timestamp) {
		r.signal = nil
		return
	}

	if r.signal == nil {
		r.signal = &models.Signal{
			Bar: bar,
			CanTradeFn: func(signal *models.Signal, direction models.Direction) bool {
				return true
			},
		}
		r.AddSignal(r.signal)
	}

	const target = 0.5

	if bar.Bullish() {
		r.CreateOrder(
			r.Symbol(),
			r.signal,
			models.OpenReasonSignal,
			models.Long,
			bar.Close,
			bar.Low,
			bar.High+target)
	} else {
		r.CreateOrder(
			r.Symbol(),
			r.signal,
			models.OpenReasonSignal,
			models.Short,
			bar.Close,
			bar.High,
			bar.Low-target)
	}
}

func (r *Test) OnTick(tick *models.Tick) {
	r.AggregateTick(tick)
}

func (r *Test) Symbol() models.Symbol {
	return models.Symbol{
		MarketID: 17068,
		QuoteID:  6374,
	}
}

type Test struct {
	*MarketContext
	active bool
	signal *models.Signal
	tz     *time.Location
}

func NewTest() *Test {
	rv := &Test{
		MarketContext: NewMarketContext(time.Minute * 1),
	}
	rv.onNewBarCb = rv.On1MinBar
	rv.onNewTickCb = rv.OnTick
	return rv
}
