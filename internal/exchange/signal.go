package exchange

import (
	"math"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

const (
	TargetPoints = 200
	MaxStopPts   = 50
	MinStopPts   = 20
)

type Signals []*Signal

type Signal struct {
	Bar           *models.Bar
	Trades        []*Trade
	CanTradeFn    func(*Signal, models.Direction) bool `json:"-"`
	TryMaxStop    bool
	EnableSmaExit bool
}

// High returns higher signal breakout with increasing number of trades
func (s *Signal) High() float64 {
	return s.Bar.High + 3.0 + (float64(len(s.Trades)) * 2.0)
}

// Low returns lower signal breakout with increasing number of trades
func (s *Signal) Low() float64 {
	return s.Bar.Low - 3.0 - (float64(len(s.Trades)) * 2.0)
}

func (s *Signal) CanTrade(direction models.Direction) bool {
	if s.CanTradeFn != nil {
		return s.CanTradeFn(s, direction)
	}
	return len(s.Trades) == 0
}

func (s *Signal) EndsAt() time.Time {
	// subtract the 5 minute duration of the starting bar or we end up with too big an end time
	return s.Bar.EndTime()
}

func (s *Signal) AddPosition(trade *Trade) {
	s.Trades = append(s.Trades, trade)
}

func (s *Signal) EncodeableClone() *Signal {
	newTrades := make([]*Trade, len(s.Trades))
	for i, t := range s.Trades {
		newTrades[i] = &(*t)
		t.Signal = nil
	}
	return &Signal{
		Bar:    s.Bar.Copy(),
		Trades: newTrades,
	}
}

func (s *Signal) EST(direction models.Direction) (entry, stop, target float64) {
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
