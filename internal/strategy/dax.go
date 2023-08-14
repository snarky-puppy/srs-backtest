package strategy

import (
	"log"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

func (d *Dax) Location() *time.Location {
	return d.Timezone()
}

func (d *Dax) IsOpen(t time.Time) bool {
	t = t.In(d.Timezone())
	return t.After(d.MarketOpen(t)) && t.Before(d.MarketClose(t))
}

func (d *Dax) MarketOpen(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, d.Location())
}

// MarketClose returns the time the market closes on day t
func (d *Dax) MarketClose(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, d.Location())
}

func (d *Dax) Timezone() *time.Location {
	if d.tz == nil {
		var err error
		d.tz, err = time.LoadLocation("Europe/Berlin")
		if err != nil {
			panic(err)
		}
	}
	return d.tz
}

func (d *Dax) isPeriod(t time.Time) bool {
	t = t.In(d.Timezone())
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}

func (d *Dax) On5MinBar() {
	bar := d.history.GetBar(0)

	if bar == nil {
		return
	}

	t := bar.Timestamp.In(d.Timezone())

	if !d.IsOpen(bar.Timestamp) {
		// don't go over closing time
		if len(d.orders) > 0 {
			for id := range d.orders {
				d.exchange.CancelOrder(id)
			}
			d.orders = make(map[int]*models.Trade)
		}
		if len(d.positions) > 0 {
			for id, trade := range d.positions {
				trade.Position = d.exchange.ExitPosition(id)
				trade.ExitReason = models.ExitReasonMarketClose
			}
			d.positions = make(map[int]*models.Trade)
		}
		return
	}

	// setup srs signal
	if d.isPeriod(t) {

		// take last 3 elements from d.bars
		const historySize = 3
		setupBars := d.history.GetBars(historySize)
		if len(setupBars) != historySize {
			panic("expected n bars")
		}
		if setupBars[0].Timestamp.Before(d.MarketOpen(t)) {
			log.Println("skipping signal due to gap in data", setupBars[0].Timestamp, setupBars[1].Timestamp, setupBars[2].Timestamp)
			return
		}
		signalBar := setupBars[0].Copy()
		for _, b := range setupBars[1:] {
			signalBar.Add(b)
		}
		signalBar.Duration = historySize * 5 * time.Minute
		d.signal = &models.Signal{
			Bar: signalBar,
			CanTradeFn: func(signal *models.Signal, direction models.Direction) bool {
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
		d.AddSignal(d.signal)

		createOrder := func(direction models.Direction) {
			entry, stop, _ := d.signal.EST(direction)
			d.CreateOrder(
				d.Symbol(),
				d.signal,
				models.OpenReasonSignal,
				direction,
				entry,
				stop,
				0)
		}

		// add 2 trades
		createOrder(models.Long)
		createOrder(models.Short)
	}

	if d.signal == nil {
		return
	}

	d.MarketContext.Common5MinBarHandler(bar)
}

func (d *Dax) OnTick(tick *models.Tick) {
	d.DefaultOnTick(tick)
}

func (d *Dax) Symbol() models.Symbol {
	return models.Symbol{
		MarketID: 17068,
		QuoteID:  6374,
	}
}

type Dax struct {
	*MarketContext
	active bool
	signal *models.Signal
	tz     *time.Location
}

func NewDax() *Dax {
	rv := &Dax{
		MarketContext: NewMarketContext(time.Minute * 5),
	}
	rv.onNewBarCb = rv.On5MinBar
	rv.onNewTickCb = rv.OnTick
	return rv
}
