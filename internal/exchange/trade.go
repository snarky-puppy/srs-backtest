package exchange

import (
	"fmt"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

type OpenReason string

const (
	OpenReasonSignal        OpenReason = "signal"
	OpenReasonAddToPosition OpenReason = "add"
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
	TrailStopPoints   float64
}

func (t *Trade) updateClosed(exTrade *ExTrade) {
	t.ExTrade = exTrade
	if t.ExitReason == "" {
		if t.Profit > 0 {
			t.ExitReason = ExitReasonTarget
		} else {
			t.ExitReason = ExitReasonStopLoss
		}
	}
}

func (t *Trade) PlotStopLine(data Series) (stopLine []opts.KlineData) {
	var (
		stopIdx int
		curStop string
	)

	curStop = "-"

	for _, bar := range data {
		if stopIdx < len(t.StopLog) && t.StopLog[stopIdx].Timestamp.Truncate(bar.Duration) == bar.Timestamp {
			curStop = fmt.Sprintf("%0.2f", t.StopLog[stopIdx].Stop)
			stopIdx++
		} else if t.ExitTime.Before(bar.Timestamp) {
			curStop = "-"
		}

		stopLine = append(stopLine, opts.KlineData{Value: curStop})
	}

	return
}
