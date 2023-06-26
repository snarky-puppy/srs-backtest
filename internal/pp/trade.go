package pp

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

type Direction int

func (d Direction) String() string {
	switch d {
	case Long:
		return "long"
	case Short:
		return "short"
	default:
		panic(fmt.Sprintf("invalid direction: %d", d))
	}
}

const (
	Long Direction = iota
	Short
)

type StopLog struct {
	Stop      float64
	Idx       int
	Timestamp time.Time
}

type Trade struct {
	Direction               Direction
	Stop                    float64
	StopPoints              float64 // how many points the stop is from the open, always positive
	Open                    float64
	OpenAt                  time.Time
	OpenAtBar               *Bar
	Close                   float64
	CloseAt                 time.Time
	CloseAtBar              *Bar
	CloseAtIdx              int
	High                    float64 // highest (or lowest if short) price reached
	HighAt                  time.Time
	HighAfterBars           int     // number of bars after open that the high was reached
	Target                  float64 // official target
	TargetAtBars            int     // number of bars after open that the target (30pt) was reached
	PostTargetHigh          float64 // highest price reached after target was reached
	PostTargetHighAfterBars int     // number of bars after target that the post target high was reached
	BarCnt                  int     // number of bars this trade was open
	StopLog                 []*StopLog
	CloseReason             string
	Loser                   float64 // the Loser score
	MaxLoser                float64 // the highest loser score during this trade
	Signal                  *Signal // the originating signal
}

func (t *Trade) Profit() float64 {
	if t.Direction == Long {
		return t.Close - t.Open
	} else {
		return t.Open - t.Close
	}
}

func (t *Trade) IsStoppedOut(b Bar) bool {
	if t.Direction == Long {
		return b.Low < t.Stop
	} else {
		return b.High > t.Stop
	}
}

func (t *Trade) WasStoppedOut() bool {
	return t.Close == t.Stop
}

func (t *Trade) PlotStopLine(bars Series) (stopLine []opts.KlineData) {
	var (
		stopIdx = 0
		curStop float64
	)

	for i := range bars {
		switch {
		case stopIdx == 0 && i+1 < len(bars) && bars[i+1].Timestamp.Equal(t.StopLog[stopIdx].Timestamp):
			// add an extra stop line for the open
			stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", t.StopLog[stopIdx].Stop)})
		case stopIdx < len(t.StopLog) && bars[i].Timestamp.Equal(t.StopLog[stopIdx].Timestamp):
			curStop = t.StopLog[stopIdx].Stop
			stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", curStop)})
			stopIdx++
		case stopIdx > 0 && stopIdx < len(t.StopLog):
			stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", curStop)})
		default:
			stopLine = append(stopLine, opts.KlineData{Value: "-"})
		}
	}
	return
}

func (t *Trade) UpdateLoserScore(l float64) {
	t.Loser += l
	t.MaxLoser = math.Max(t.MaxLoser, t.Loser)
}

func (t *Trade) IsLoser() bool {
	return t.Loser >= 5
}

func (t *Trade) ResetLoserStatus() {
	t.Loser = 0
}

func (t *Trade) IsLoserStopMsg() string {
	if t.IsLoser() {
		return " (loser)"
	}
	return ""
}

func (t *Trade) TradeCloseMsg(raison string) string {
	raisons := []string{}
	if t.IsLoser() {
		raisons = append(raisons, "loser")
	}

	if len(raisons) > 0 {
		return fmt.Sprintf("%s (%s)", raison, strings.Join(raisons, ","))
	}
	return raison
}

func (t *Trade) CheckLoser(bar *Bar) {
	/*
		1. Close losing trades early: if the trade is in loss for too long, 3-5 bars in, close the trade
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
	isStraddle := bar.High > t.Open && bar.Low < t.Open
	isSunken := false
	switch t.Direction {
	case Long:
		isSunken = bar.High < t.Open
	case Short:
		isSunken = bar.Low > t.Open
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

	if t.IsLoser() {
		switch t.Direction {
		case Long:
			t.Target = t.Open
			t.Stop = bar.Low + 3
		case Short:
			t.Target = t.Open
			t.Stop = bar.High + 3
		}
	}
}
