package models

import (
	"time"
)

type Signal struct {
	Bar           *Bar
	Trades        []*Trade
	CanTradeFn    func(*Signal, Direction) bool `json:"-"`
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

func (s *Signal) CanTrade(direction Direction) bool {
	if s.CanTradeFn != nil {
		return s.CanTradeFn(s, direction)
	}
	return len(s.Trades) == 0
}

func (s *Signal) EndsAt() time.Time {
	// subtract the 5 minute duration of the starting bar or we end up with too big an end time
	return s.Bar.EndTime()
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

func (s *Signal) AddPosition(trade *Trade) {
	s.Trades = append(s.Trades, trade)
}
