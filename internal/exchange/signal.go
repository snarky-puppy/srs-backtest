package exchange

import (
	"time"
)

const (
	TargetPoints = 20
)

type Signals []*Signal

type Signal struct {
	Bar        *Bar
	Trades     []*Trade
	CanTradeFn func(*Signal, Direction) bool
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
	return s.Bar.Timestamp.Add(s.Bar.Duration - (5 * time.Minute))
}

//func (s *Signal) NewTrade(direction Direction) *Trade {
//	return NewTrade(direction, s.Entry(direction), s.Stop(direction), s.Target(direction), 0, nil, s, "srs crossover")
//}

func (s *Signal) Target(direction Direction) float64 {
	switch direction {
	case Long:
		return s.Entry(direction) + TargetPoints
	case Short:
		return s.Entry(direction) - TargetPoints
	default:
		panic("invalid direction")
	}
}

func (s *Signal) Stop(direction Direction) float64 {
	// win rate 33-36%
	// total profit: 10555
	//return (bar.High + bar.Low) / 2

	// win rate: 41-44%
	// total profit: 12257
	if direction == Long {
		return s.Bar.Low
	}
	return s.Bar.High
}

func (s *Signal) Entry(direction Direction) float64 {
	switch direction {
	case Long:
		return s.High()
	case Short:
		return s.Low()
	}
	panic("invalid direction")
}

func (s *Signal) AddPosition(trade *Trade) {
	s.Trades = append(s.Trades, trade)
}
