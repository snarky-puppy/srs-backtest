package pp

import (
	"compress/gzip"
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

type Series []*Bar

func (s Series) Resample(d time.Duration) Series {
	resampledBars := Series{}
	var open, high, low, cls float64
	var intervalStart time.Time

	for i, bar := range s {
		if i == 0 {
			// Initialize for the first bar
			open, high, low, cls = bar.Open, bar.High, bar.Low, bar.Close
			intervalStart = bar.Timestamp.Truncate(d)
			continue
		}

		if bar.Timestamp.Truncate(d) != intervalStart {
			// Start of a new interval, append the accumulated bar to resampledBars
			resampledBars = append(resampledBars, &Bar{Timestamp: intervalStart, Open: open, High: high, Low: low, Close: cls})

			// Reset for the new interval
			open, high, low, cls = bar.Open, bar.High, bar.Low, bar.Close
			intervalStart = bar.Timestamp.Truncate(d)
		} else {
			// Still the same interval, update high and low
			if bar.High > high {
				high = bar.High
			}
			if bar.Low < low {
				low = bar.Low
			}
			cls = bar.Close
		}
	}

	// Don't forget the last interval
	resampledBars = append(resampledBars, &Bar{Timestamp: intervalStart, Open: open, High: high, Low: low, Close: cls})

	return resampledBars
}

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

func (s Series) FilterYear(year int) (rv Series) {
	for _, bar := range s {
		if bar.Timestamp.Year() == year {
			rv = append(rv, bar)
		}
	}
	return rv
}

// There is a trade at next, get the bars before and after it
func (s Series) SignalContext(signal *Signal) (idx int, rv Series) {
	start := signal.Idx - 20
	var end int
	if len(signal.Trades) > 0 {
		for _, t := range signal.Trades {
			if t.CloseAtIdx > end {
				end = t.CloseAtIdx
			}
		}
	} else {
		end = start + 40
	}
	end += 20
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	for i := start; i < end; i++ {
		//if s[i].MarketCloseBar() && i > ((start+end)/2) {
		//	break
		//}
		rv = append(rv, s[i])
	}
	return start, rv
}

func (s Series) SignalContext2(signal *Signal) (rv Series) {
	start := signal.Bar.Timestamp.Add(-(2 * signal.Bar.Duration))
	var end = start.Add(1 * time.Hour)
	if len(signal.Trades) > 0 {
		for _, t := range signal.Trades {
			if t.CloseTime.After(end) {
				end = t.CloseTime.Add(30 * time.Minute)
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

func ReadBarsFromCSV(filename string, useLocalTz bool) (Series, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	reader := csv.NewReader(gzr)

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	bars := make([]*Bar, len(lines)-1)

	for i, line := range lines[1:] { // Skip header
		timestamp, err := time.Parse(time.RFC3339, line[0]) // Parse timestamp in RFC3339 format
		if err != nil {
			return nil, err
		}

		if useLocalTz {
			timestamp = timestamp.Local()
		}

		open, err := strconv.ParseFloat(line[1], 32)
		if err != nil {
			return nil, err
		}

		high, err := strconv.ParseFloat(line[2], 32)
		if err != nil {
			return nil, err
		}

		low, err := strconv.ParseFloat(line[3], 32)
		if err != nil {
			return nil, err
		}

		closePrice, err := strconv.ParseFloat(line[4], 32)
		if err != nil {
			return nil, err
		}

		bars[i] = &Bar{
			Timestamp: timestamp,
			Open:      float64(open),
			High:      float64(high),
			Low:       float64(low),
			Close:     float64(closePrice),
		}
	}

	return bars, nil
}
