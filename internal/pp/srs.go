package pp

import (
	"fmt"
)

type SRS struct {
	srsHigh float64
	srsLow  float64

	Trades      []Trade
	activeTrade *Trade

	prevState TradingState
	prevBar   *Bar
}

func (s *SRS) NextBar(b Bar) {

	defer func() {
		s.prevBar = &b
	}()

	state := NewTradingState(b.Timestamp)
	if state != s.prevState {
		if s.prevState == Trading {
			// reset srsHigh and srsLow for tomorrow
			s.srsHigh = 0
			s.srsLow = 0
		}
		s.prevState = state
	}
	switch state {
	case Scanning:
		s.scanning(b)
	case Trading:
		s.trading(b)
	case Inactive:
		if s.activeTrade != nil {
			s.maintainActiveTrade(b)
		}
	}
}

func (s *SRS) scanning(b Bar) {
	if s.srsLow == 0 || b.Low < s.srsLow {
		s.srsLow = b.Low
	}
	if s.srsHigh == 0 || b.High > s.srsHigh {
		s.srsHigh = b.High
	}
}

func (s *SRS) trading(b Bar) {
	if s.activeTrade != nil {
		s.maintainActiveTrade(b)
	}

	// TODO: check if today is flat
	// TODO: check if this month or week is trending

	// if this bar crosses over srsHigh, open a trade
	if s.prevBar != nil && s.prevBar.High < s.srsHigh && b.Low < s.srsHigh && b.High > s.srsHigh {
		s.activeTrade = &Trade{
			OpenPrice: s.srsHigh + 3,
			StopPrice: s.srsLow,
			Direction: Long,
			OpenTime:  b.Timestamp,
		}
	}
	// if this bar crosses under srsLow, open a trade
	if s.prevBar != nil && s.prevBar.Low > s.srsLow && b.High > s.srsLow && b.Low < s.srsLow {
		s.activeTrade = &Trade{
			OpenPrice: s.srsLow - 3,
			StopPrice: s.srsHigh,
			Direction: Short,
			OpenTime:  b.Timestamp,
		}
	}
}

func (s *SRS) maintainActiveTrade(b Bar) {
	if s.activeTrade.IsStoppedOut(b) {
		s.activeTrade.CloseTime = b.Timestamp
		s.activeTrade.ClosePrice = s.activeTrade.StopPrice
		s.Trades = append(s.Trades, *s.activeTrade)
		s.activeTrade = nil
	}
	// take profit at 20 points
	if s.activeTrade != nil && s.activeTrade.Direction == Long && b.High-s.activeTrade.OpenPrice > 20 {
		s.activeTrade.CloseTime = b.Timestamp
		s.activeTrade.ClosePrice = s.activeTrade.OpenPrice + 20
		s.Trades = append(s.Trades, *s.activeTrade)
		s.activeTrade = nil
	}
	if s.activeTrade != nil && s.activeTrade.Direction == Short && s.activeTrade.OpenPrice-b.Low > 20 {
		s.activeTrade.CloseTime = b.Timestamp
		s.activeTrade.ClosePrice = s.activeTrade.OpenPrice - 20
		s.Trades = append(s.Trades, *s.activeTrade)
		s.activeTrade = nil
	}
}

func (s *SRS) PrintStats() {
	var (
		stops      int
		profitable int
		profit     float64
	)

	for _, t := range s.Trades {
		p := t.Profit()
		profit += p
		if t.WasStoppedOutForZero() {
			stops++
		} else {
			profitable++
		}
	}

	fmt.Printf("%d trades\n", len(s.Trades))
	fmt.Printf("%s to %s\n", s.Trades[0].OpenTime, s.Trades[len(s.Trades)-1].CloseTime)
	fmt.Printf("Percent win: %f\n", float64(profitable)/float64(len(s.Trades)))
	fmt.Printf("Stopped out: %d\n", stops)
	fmt.Printf("Total profit: %f\n", profit)
	//for _, t := range s.Trades {
	//	println(t.OpenPrice, t.ClosePrice, t.StopPrice, t.Direction, t.ClosePrice-t.OpenPrice)
	//}
}

func NewSRS() *SRS {
	return &SRS{}
}
