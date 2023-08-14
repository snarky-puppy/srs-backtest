package models

import (
	"fmt"
	"math"
	"time"

	"github.com/mwlazlo/srs/internal"
)

type OpenReason string

const (
	OpenReasonSignal        OpenReason = "signal"
	OpenReasonAddToPosition OpenReason = "add"
)

type ExitReason string

const (
	ExitReasonTarget       ExitReason = "target"
	ExitReasonStopLoss     ExitReason = "stoploss"
	ExitReasonMarketClose  ExitReason = "marketclose"
	ExitReasonSmaCross     ExitReason = "sma"
	ExitReasonSmaCrossStop ExitReason = "smastop"
)

// StopLog tracks stop adjustments
type StopLog struct {
	Stop      float64
	Timestamp time.Time
}

type Trade struct {
	*Position

	StopLog           []*StopLog
	OpenReason        OpenReason
	ExitReason        ExitReason
	Loser             float64 // the Loser score
	DisableLoserCheck bool    // disable the Loser score (for add-on trades)
	Signal            *Signal // the originating signal
	AutoAdjustStop    bool    // if true, trade management will automatically update this trade's stop
	CanAddToPosition  bool
	IsAdditional      bool
	TrailStopPoints   float64
	LoserThreshold    float64
	BTL               int // Bars To Live (close when BTL == 0)
}

func (t *Trade) UpdateClosed(exTrade *Position) {
	if t.ExitReason == "" {
		if math.Abs(t.ExitPrice-t.StopPrice) <= 0.01 {
			t.ExitReason = ExitReasonStopLoss
		}
	}
}

func (t *Trade) PlotStopLine(data Series) (stopLine []string) {
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

		stopLine = append(stopLine, curStop)
	}

	return
}

func (t *Trade) CheckLoser(bar *Bar) {
	/*
		1. ClosePrice losing trades early: if the trade is in loss for too long, 3-5 bars in, close the trade
			 (dubbed "sunken" - bar's high is still a loss)
			- long: the high is under the trigger for 3 bars -- close
			- short: the low is over the trigger for 3 bars -- close
		or
			(dubbed "straddle")
			- long: high above and low below the trigger for 4 consecutive -- close
			- short: low below and high above the trigger for 4 consecutive -- close
		or
			(dubbed modified straddle)
			- straddle for 3 then move out -- set stop to break even
		or
			- long: straddle for 3 then high below the trigger for 1 -- close
			- short: straddle for 3 then low above the trigger for 1 -- close

		Loser score:
		  + 1.5 for every bar high is under trigger
		  + 1   for every bar straddle
		  ** RESET score when 5 min bar clears the trigger (long: low > trigger, short: high < trigger)
	*/

	if t.DisableLoserCheck {
		return
	}

	isStraddle := bar.High > t.OpenPrice && bar.Low < t.OpenPrice
	isSunken := false
	switch t.Direction {
	case Long:
		isSunken = bar.High < t.OpenPrice
	case Short:
		isSunken = bar.Low > t.OpenPrice
	}

	if isSunken && isStraddle {
		panic("failed sanity check: isSunken && isStraddle")
	}

	switch {
	case isSunken:
		t.UpdateLoserScore(1.5)
	case isStraddle:
		t.UpdateLoserScore(1)
	default:
		// Once the trade is a loser, it can't be recovered because the stop/break even trades are already submitted
		if !t.IsLoser() /* || LoserCanRecover */ {
			t.ResetLoserStatus()
		}
	}
}

func (t *Trade) UpdateLoserScore(l float64) {
	t.Loser += l
}

func (t *Trade) IsLoser() bool {
	return t.Loser >= t.LoserThreshold
}

func (t *Trade) ResetLoserStatus() {
	t.Loser = 0
}

func CalculatePointsProfit(t *Position, close float64) float64 {
	switch t.Direction {
	case Long:
		return internal.Round2(close - t.EntryPrice)
	case Short:
		return internal.Round2(t.EntryPrice - close)
	}
	return 0
}

func CalculateRealProfit(t *Position, close float64) float64 {
	return internal.Round2(t.Size * CalculatePointsProfit(t, close))
}
