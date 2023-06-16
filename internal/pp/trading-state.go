package pp

import (
	"fmt"
	"time"
)

type TradingState int

const (
	Inactive TradingState = iota
	Scanning
	Trading
)

func (t TradingState) String() string {
	switch t {
	case Scanning:
		return "Scanning"
	case Trading:
		return "Trading"
	case Inactive:
		return "Inactive"
	default:
		return fmt.Sprintf("Unknown state: %d", t)
	}
}

func NewTradingState(t time.Time) TradingState {
	if t.Hour() >= 8 && t.Hour() <= 11 {
		if t.Hour() == 8 {
			if t.Minute() < 15 {
				return Inactive
			}
			if t.Minute() >= 15 && t.Minute() < 30 {
				return Scanning
			}
			if t.Minute() >= 30 {
				return Trading
			}
		} else {
			return Trading
		}
	}
	return Inactive
}
