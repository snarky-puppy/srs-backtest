package strategy

import (
	"log"
	"math"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

func (d *DaxAggro) Location() *time.Location {
	return d.Timezone()
}

func (d *DaxAggro) IsOpen(t time.Time) bool {
	t = t.In(d.Timezone())
	return t.After(d.MarketOpen(t)) && t.Before(d.MarketClose(t))
}

func (d *DaxAggro) MarketOpen(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, d.Location())
}

// MarketClose returns the time the market closes on day t
func (d *DaxAggro) MarketClose(t time.Time) time.Time {
	t = t.In(d.Timezone())
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, d.Location())
}

func (d *DaxAggro) Timezone() *time.Location {
	if d.tz == nil {
		var err error
		d.tz, err = time.LoadLocation("Europe/Berlin")
		if err != nil {
			panic(err)
		}
	}
	return d.tz
}

func (d *DaxAggro) isPeriod(t time.Time) bool {
	t = t.In(d.Timezone())
	if t.Minute() == 25 && t.Hour() == 9 {
		return true
	}
	return false
}

func (d *DaxAggro) On5MinBar() {
	bar := d.history.GetBar(0)

	if bar == nil {
		return
	}

	barTs := bar.Timestamp.In(d.Timezone())

	if !d.IsOpen(bar.Timestamp) {
		d.CloseAll("")
		return
	}

	// setup srs signal
	if d.isPeriod(barTs) {
		d.setupSignal(barTs)
	}

	if d.signal == nil {
		return
	}

	for _, position := range d.positions {
		position.CheckLoser(bar)
		if position.IsLoser() {
			// try to close for beak-even
			switch position.Direction {
			case models.Long:
				d.exchange.UpdatePosition(position.Id, math.Max(position.StopPrice, bar.Low+3), position.TargetPrice)
			case models.Short:
				d.exchange.UpdatePosition(position.Id, math.Min(position.StopPrice, bar.High+3), position.TargetPrice)
			}
			continue
		}

		// if we are long, and the 25 SMA crosses below the 5 SMA, exit the position
		if position.Signal.EnableSmaExit {
			if position.Direction == models.Long {
				if d.history.Sma25.CrossedOver(d.history.Sma5, 2) {
					if models.CalculatePointsProfit(position.Position, bar.Close) <= 0 {
						// if not in profit, tighten the stop and try to close for break even
						stop := d.history.FindAverageLow(5)
						position.ExitReason = models.ExitReasonSmaCrossStop
						d.UpdatePosition(position, stop, position.EntryPrice)
					} else {
						// otherwise, just close for profit
						position.ExitReason = models.ExitReasonSmaCross
						d.ExitPosition(position.Id)
					}
					continue
				}
			} else {
				if d.history.Sma5.CrossedOver(d.history.Sma25, 2) {
					if models.CalculatePointsProfit(position.Position, bar.Close) <= 0 {
						// if not in profit, tighten the stop and try to close for break even
						stop := d.history.FindAverageHigh(5)
						position.ExitReason = models.ExitReasonSmaCrossStop
						d.UpdatePosition(position, stop, position.EntryPrice)
					} else {
						// otherwise, just close for profit
						position.ExitReason = models.ExitReasonSmaCross
						d.ExitPosition(position.Id)
					}
					continue
				}
			}
		}

		if position.CanAddToPosition {
			d.considerAddingToPosition(bar, position)
		}
	}

}

func (d *DaxAggro) tickUpdateStop(tick *models.Tick) {
	for _, trade := range d.positions {
		tickPrice := tick.MidPrice()

		if trade.AutoAdjustStop {
			switch trade.Direction {
			case models.Long:
				// trail by 30 pts
				if (tickPrice - trade.TrailStopPoints) > trade.StopPrice {
					d.UpdatePosition(trade, tickPrice-trade.TrailStopPoints, 0)
				}
			case models.Short:
				if (tickPrice + trade.TrailStopPoints) < trade.StopPrice {
					d.UpdatePosition(trade, tickPrice+trade.TrailStopPoints, 0)
				}
			}
		}
	}
}

func (d *DaxAggro) considerAddingToPosition(bar *models.Bar, winner *models.Trade) {

	if winner.EntryTime.Truncate(bar.Duration).Equal(bar.Timestamp) {
		return
	}

	if models.CalculatePointsProfit(winner.Position, bar.AvgPrice()) < AddToPositionPoints {
		return
	}

	// crude velocity check
	switch winner.Direction {
	case models.Long:
		if d.history.Sma5.Calculate()-d.history.Sma25.Calculate() < AddToTradeVelocityThreshold {
			return
		}
	case models.Short:
		if d.history.Sma25.Calculate()-d.history.Sma5.Calculate() < AddToTradeVelocityThreshold {
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

	newTrade := d.CreateOrder(winner.Symbol, winner.Signal, models.OpenReasonAddToPosition, winner.Direction, open, winner.StopPrice, 0)
	newTrade.IsAdditional = true
	//newTrade.BTL = 5
}

func (d *DaxAggro) OnTick(tick *models.Tick) {
	bar := d.AggregateTick(tick)
	if bar == nil {
		d.On5MinBar()
	}
	d.tickUpdateStop(tick)
}

func (d *DaxAggro) Symbol() models.Symbol {
	return models.Symbol{
		MarketID:   17068,
		QuoteID:    6374,
		MarketName: "Germany 40 - Rolling Cash",
	}
}

func (d *DaxAggro) setupSignal(tm time.Time) {
	// take last 3 elements from d.bars
	const historySize = 3
	setupBars := d.history.GetBars(historySize)
	if len(setupBars) != historySize {
		panic("expected n bars")
	}
	if setupBars[0].Timestamp.Before(d.MarketOpen(tm)) {
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
		entry, stop, _ := d.EST(direction)
		o := d.CreateOrder(
			d.Symbol(),
			d.signal,
			models.OpenReasonSignal,
			direction,
			entry,
			stop,
			0)
		o.AutoAdjustStop = true
		o.LoserThreshold = 15
		o.CanAddToPosition = true
	}

	// add 2 trades
	createOrder(models.Long)
	createOrder(models.Short)
}

func (d *DaxAggro) EST(direction models.Direction) (entry, stop, target float64) {
	const (
		TargetPoints = 200
		MaxStopPts   = 50
		MinStopPts   = 20
	)

	s := d.signal
	switch direction {
	case models.Long:
		entry = s.High() + 3
		target = entry + TargetPoints

		if s.TryMaxStop {
			stop = math.Max(s.Bar.Low, entry-MaxStopPts)
		} else {
			stopPts := s.Bar.High - s.Bar.Low
			if stopPts > MaxStopPts {
				stop = s.Bar.High - MaxStopPts
			} else if stop < MinStopPts {
				stop = s.Bar.High - MinStopPts
			} else {
				stop = s.Bar.Low
			}
		}
	case models.Short:
		entry = s.Low() - 3
		target = entry - TargetPoints

		if s.TryMaxStop {
			stop = math.Min(s.Bar.High, entry+MaxStopPts)
			break
		} else {
			stopPts := s.Bar.High - s.Bar.Low
			if stopPts > MaxStopPts {
				stop = s.Bar.Low + MaxStopPts
			} else if stop < MinStopPts {
				stop = s.Bar.Low + MinStopPts
			} else {
				stop = s.Bar.High
			}
		}
	}
	return
}

type DaxAggro struct {
	*MarketContext
	active bool
	signal *models.Signal
	tz     *time.Location
}

func NewDaxAggro() *DaxAggro {
	rv := &DaxAggro{
		MarketContext: NewMarketContext(time.Minute * 5),
	}
	rv.onNewBarCb = rv.On5MinBar
	rv.onNewTickCb = rv.OnTick
	return rv
}
