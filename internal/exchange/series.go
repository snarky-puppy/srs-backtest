package exchange

import (
	"math"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

type Series []*Bar

func (s Series) ToChartXAxis() (rv []time.Time) {
	for _, bar := range s {
		rv = append(rv, bar.Timestamp)
	}
	return rv
}

func (s Series) ToChartData() (rv []opts.KlineData) {
	for _, bar := range s {
		rv = append(rv, opts.KlineData{
			Value: []float64{bar.Open, bar.Close, bar.Low, bar.High},
		})
	}
	return rv
}

func (s Series) SignalContext(signal *Signal) (rv Series) {
	start := signal.Bar.Timestamp.Add(-(2 * signal.Bar.Duration))
	var end = start.Add(1 * time.Hour)
	if len(signal.Trades) > 0 {
		for _, t := range signal.Trades {
			if t.ExitTime.After(end) {
				end = t.ExitTime.Add(30 * time.Minute)
			}
		}
	}

	// find index of start, use binary search
	startIdx := sort.Search(len(s), func(i int) bool {
		return s[i].Timestamp.After(start)
	})
	endIdx := sort.Search(len(s), func(i int) bool {
		return s[i].Timestamp.After(end)
	})
	return s[startIdx:endIdx]
}

func (s Series) FilterDay(day time.Time) Series {
	var rv Series
	for _, bar := range s {
		t := bar.Timestamp.In(day.Location())
		if t.Year() == day.Year() && t.Month() == day.Month() && t.Day() == day.Day() {
			rv = append(rv, bar)
		}
	}
	sort.Slice(rv, func(i, j int) bool {
		return rv[i].Timestamp.Before(rv[j].Timestamp)
	})
	return rv
}

func (s Series) HeadIdx() int {
	return len(s) - 1
}

func (s Series) Head() *Bar {
	return s.GetBar(s.HeadIdx())
}

func (s Series) GetBar(offset int) *Bar {
	if len(s) == 0 || int(math.Abs(float64(offset))) > len(s) {
		return nil
	}
	return s[s.HeadIdx()+offset]
}
