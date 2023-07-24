package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/mwlazlo/srs/internal/exchange"
)

const DATA = "data/td/Germany 40 - Rolling Cash.csv.gz"

func main() {
	reader, err := exchange.NewTickReader(DATA)
	if err != nil {
		panic(err)
	}

	agg := exchange.NewBarAggregator(5 * time.Minute)

	var bars []*exchange.Bar
	for {
		tick, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		if tick == nil {
			break
		}
		bar := agg.ProcessTick(tick)
		if bar != nil {
			bars = append(bars, bar)
		}
	}

	fmt.Println("opening http://localhost:8081/")
	http.HandleFunc("/", endpoint(bars))
	_ = http.ListenAndServe("localhost:8081", nil)

}

func getDay(dayStr string, bars []*exchange.Bar) time.Time {
	if dayStr == "" {
		t := bars[0].Timestamp
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	day, err := time.ParseInLocation("2006-01-02", dayStr, bars[0].Timestamp.Location())
	if err != nil {
		fmt.Println(err)
	}
	return day
}

func endpoint(bars exchange.Series) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day := getDay(r.URL.Query().Get("d"), bars)
		data := bars.FilterDay(day)
		// create a new line instance
		line := charts.NewKLine()

		line.SetXAxis(data.ToChartXAxis()).
			AddSeries("", data.ToChartData())

		// set some global options like Title/Legend/ToolTip or anything else
		line.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
			charts.WithTitleOpts(opts.Title{
				Title: fmt.Sprintf("%s",
					day.Format("2006-01-02"),
				),
			}),
			charts.WithYAxisOpts(opts.YAxis{
				Scale: true,
				SplitArea: &opts.SplitArea{
					Show: true,
				},
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "inside",
				Start: 0,
				End:   100,
			}),
			charts.WithDataZoomOpts(opts.DataZoom{
				Type:  "slider",
				Start: 0,
				End:   100,
			}),
			charts.WithTooltipOpts(opts.Tooltip{
				Show:      true,
				Trigger:   "axis",
				TriggerOn: "mousemove|click",
				AxisPointer: &opts.AxisPointer{
					Type: "cross",
				},
			}),
		)
		_ = line.Render(w)

		// generate prev and next links
		prev := day.Add(-24 * time.Hour).Format("2006-01-02")
		next := day.Add(24 * time.Hour).Format("2006-01-02")
		_, _ = fmt.Fprintf(w, "<a href='/?d=%s'>prev</a> <a href='/?d=%s'>next</a>", prev, next)
	}
}
