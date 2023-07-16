package pp

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

const (
	MaxStop = 100
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

// StopLog tracks the stop price and the bar index at which it was set
type StopLog struct {
	Stop      float64
	Idx       int
	Timestamp time.Time
}

func NewTrade(direction Direction, open, stop, target float64, index int, bar *Bar, signal *Signal, reason string) *Trade {
	// off: total profit: 20062
	// on:  total profit: 19967
	//switch direction {
	//case Long:
	//	if open-stop > MaxStop {
	//		fmt.Println(open-stop, "stop too big")
	//		stop = open - MaxStop
	//	}
	//case Short:
	//	if stop-open > MaxStop {
	//		fmt.Println(open-stop, "stop too big")
	//		stop = open + MaxStop
	//	}
	//}

	return &Trade{
		OpenPrice: open,
		StopPrice: stop,
		StopLog: []*StopLog{{
			Stop:      stop,
			Idx:       index,
			Timestamp: bar.Timestamp,
		}},
		Target:    target,
		Direction: direction,
		OpenAtBar: bar,
		OpenTime:  bar.Timestamp,
		OpenAtIdx: index,
		High: func() float64 {
			if direction == Long {
				return bar.High
			}
			return bar.Low
		}(),
		Signal:         signal,
		AutoAdjustStop: false,
		Reason:         reason,
	}
}

type Trade struct {
	Id                      int
	Direction               Direction
	Qty                     float64
	StopPrice               float64
	OpenPrice               float64
	OpenTime                time.Time
	OpenAtBar               *Bar
	OpenAtIdx               int
	ClosePrice              float64
	CloseTime               time.Time
	CloseAtBar              *Bar
	CloseAtIdx              int
	High                    float64 // highest (or lowest if short) price reached
	HighAt                  time.Time
	HighAfterBars           int     // number of bars after open that the high was reached
	Target                  float64 // official target
	PostTargetHigh          float64 // highest price reached after target was reached
	PostTargetHighAfterBars int     // number of bars after target that the post target high was reached
	BarCnt                  int     // number of bars this trade was open
	StopLog                 []*StopLog
	CloseReason             string
	Loser                   float64 // the Loser score
	DisableLoserCheck       bool    // disable the Loser score (for add-on trades)
	MaxLoser                float64 // the highest loser score during this trade
	Signal                  *Signal // the originating signal
	AutoAdjustStop          bool    // if true, trade management will automatically update this trade's stop
	Reason                  string
	CanAddToPosition        bool
	IsAdditional            bool
}

func (t *Trade) Profit() float64 {
	if t.Direction == Long {
		return t.ClosePrice - t.OpenPrice
	} else {
		return t.OpenPrice - t.ClosePrice
	}
}

func (t *Trade) IsStoppedOut(b Bar) bool {
	if t.Direction == Long {
		return b.Low < t.StopPrice
	} else {
		return b.High > t.StopPrice
	}
}

func (t *Trade) WasStoppedOutForZero() bool {
	return t.ClosePrice == t.StopPrice
}

func (t *Trade) PlotStopLine(bars Series, dataIdx int) (stopLine []opts.KlineData) {
	var (
		stopIdx = 0
		curStop float64
	)

	for range bars {
		if stopIdx < len(t.StopLog) {
			stop := t.StopLog[stopIdx]
			if stop.Idx == dataIdx {
				curStop = stop.Stop
				stopIdx++
			}
		}
		if curStop == 0 {
			stopLine = append(stopLine, opts.KlineData{Value: "-"})
		} else {
			stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", curStop)})
		}
		if t.CloseAtIdx == dataIdx {
			break
		}
		dataIdx++
		/*
			switch {
			//case stopIdx == 0 && i+1 < len(bars) && bars[i+1].Timestamp.Equal(t.StopLog[stopIdx].Timestamp):
				// add an extra stop line for the open
				//stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", t.StopLog[stopIdx].StopPrice)})
			case stopIdx < len(t.StopLog) && bars[i].Timestamp.Equal(t.StopLog[stopIdx].Timestamp):
				curStop = t.StopLog[stopIdx].StopPrice
				stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", curStop)})
				stopIdx++
			case stopIdx > 0 && stopIdx < len(t.StopLog):
				stopLine = append(stopLine, opts.KlineData{Value: fmt.Sprintf("%.2f", curStop)})
			default:
				stopLine = append(stopLine, opts.KlineData{Value: "-"})
			}

		*/
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

	if t.BarCnt <= 1 { // first bar doesn't count
		return
	}

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

	if t.IsLoser() {
		switch t.Direction {
		case Long:
			t.Target = t.OpenPrice
			t.StopPrice = math.Max(t.StopPrice, bar.Low+3)
		case Short:
			t.Target = t.OpenPrice
			t.StopPrice = math.Min(t.StopPrice, bar.High+3)
		}
	}
}

func (t *Trade) AdjustStop(price float64, i int, bar *Bar) {
	t.StopPrice = price
	t.StopLog = append(t.StopLog, &StopLog{Timestamp: bar.Timestamp, Stop: price, Idx: i})
}
