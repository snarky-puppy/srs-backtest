package pp

import (
	"fmt"
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
