package exchange

import (
	"time"
)

type OpenReason string

const (
	OpenReasonSignal OpenReason = "signal"
)

type ExitReason string

const (
	ExitReasonTarget      ExitReason = "target"
	ExitReasonStopLoss    ExitReason = "stoploss"
	ExitReasonMarketClose ExitReason = "marketclose"
)

// StopLog tracks stop adjustments
type StopLog struct {
	Stop      float64
	Timestamp time.Time
}

type Trade struct {
	*ExTrade
	StopLog    []*StopLog
	OpenReason OpenReason
	ExitReason ExitReason

	Loser             float64 // the Loser score
	DisableLoserCheck bool    // disable the Loser score (for add-on trades)
	MaxLoser          float64 // the highest loser score during this trade
	Signal            *Signal // the originating signal
	AutoAdjustStop    bool    // if true, trade management will automatically update this trade's stop
	CanAddToPosition  bool
	IsAdditional      bool
	OrderTime         time.Time
}

func (t *Trade) updateClosed(exTrade *ExTrade) {
	t.ExTrade = exTrade
}
