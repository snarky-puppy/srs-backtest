package models

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

func (s Series) FilterDay(day time.Time, location *time.Location) Series {
	var rv Series
	for _, bar := range s {
		t := bar.Timestamp.In(location)
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

func (s Series) UpdateDuration(duration time.Duration) Series {
	// each s in the Series is a duration less than `duration`
	if s[0].Duration > duration {
		panic("updated duration is less than current duration")
	}
	if s[0].Duration == duration {
		return s
	}
	agg := NewBarAggregator(duration)
	rv := make([]*Bar, 0)
	for _, bar := range s {
		b := agg.AddBar(bar)
		if b != nil {
			rv = append(rv, b)
		}
	}
	final := agg.LastBar()
	if final != nil {
		rv = append(rv, final)
	}

	return rv
}
