package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/mwlazlo/srs/internal/exchange"
)

type ReportReader struct {
	baseDir string
	files   []string
}

type position struct {
	prev string
	cur  string
	next string
}

func (r *ReportReader) Report(cursor string) position {
	idx := sort.SearchStrings(r.files, cursor)
	if idx == len(r.files) {
		return r.Report(r.files[0])
	}
	if idx == 0 {
		return position{
			prev: "",
			cur:  r.files[0],
			next: r.files[1],
		}
	}
	if idx == len(r.files)-1 {
		return position{
			prev: r.files[idx-1],
			cur:  r.files[idx],
			next: "",
		}
	}
	return position{
		prev: r.files[idx-1],
		cur:  r.files[idx],
		next: r.files[idx+1],
	}
}

func (r *ReportReader) LoadRecord(cur string) (rv exchange.HistoricalRecord) {
	fp, err := os.Open(fmt.Sprintf("%s/%s", r.baseDir, cur))
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	err = json.NewDecoder(fp).Decode(&rv)
	if err != nil {
		panic(err)
	}
	return
}

func NewReportReader(baseDir string) *ReportReader {

	files, err := os.ReadDir(baseDir)
	if err != nil {
		panic(err)
	}

	var names []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			names = append(names, f.Name())
		}
	}

	sort.Strings(names)

	return &ReportReader{
		files:   names,
		baseDir: baseDir,
	}
}

func main() {
	rr := NewReportReader("data/reports")

	fmt.Println("opening http://localhost:8081/")
	http.Handle("/static/", http.FileServer(http.Dir("data/reports")))
	http.HandleFunc("/", endpoint(rr))
	_ = http.ListenAndServe("localhost:8081", nil)

}

func endpoint(rr *ReportReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the day from the query string

		pos := rr.Report(r.URL.Query().Get("d"))
		report := rr.LoadRecord(pos.cur)
		Render(w, pos, report)

		/*
			// create a new line instance
			line := charts.NewKLine()

			data := report.Context
			signal := report.Signal

			// set some global options like Title/Legend/ToolTip or anything else
			line.SetGlobalOptions(
				charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
				charts.WithTitleOpts(opts.Title{
					Title: fmt.Sprintf("%s",
						pos.cur,
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

			var opt []charts.SeriesOpts

			lArrow := func(y float64, x time.Time, label string) opts.MarkPointDataItem {
				return opts.MarkPointDataItem{
					Value: label,
					ItemStyle: &opts.ItemStyle{
						Color:       "auto",
						BorderColor: "black",
					},
					Label: &opts.Label{
						Show:     true,
						Position: "right",
					},
					Symbol:       "arrow",
					SymbolRotate: 90,
					SymbolSize:   10,
					YAxis:        y,
					XAxis:        x,
				}
			}

			//rArrow := func(y float64, x time.Time, label string) opts.MarkPointDataItem {
			//	return opts.MarkPointDataItem{
			//		Value: label,
			//		ItemStyle: &opts.ItemStyle{
			//			Color:       "auto",
			//			BorderColor: "black",
			//		},
			//		Label: &opts.Label{
			//			Show:     true,
			//			Position: "left",
			//		},
			//		Symbol:       "arrow",
			//		SymbolRotate: 180,
			//		SymbolSize:   10,
			//		YAxis:        y,
			//		XAxis:        x,
			//	}
			//}

			// region signal
			opt = append(opt,
				charts.WithMarkAreaDataItem(
					// red 15min bar
					opts.MarkAreaDataItem{
						YAxis: signal.Bar.High,
						XAxis: signal.Bar.Timestamp,
						ItemStyle: &opts.ItemStyle{
							BorderColor: "rgba(255, 0, 0, 0.3)",
							BorderWidth: 1,
						},
					},
					opts.MarkAreaDataItem{
						YAxis: signal.Bar.Low,
						XAxis: signal.EndsAt(),
					},
				),
				// grey signal line
				charts.WithMarkAreaDataItem(
					opts.MarkAreaDataItem{
						YAxis: signal.Bar.High,
						ItemStyle: &opts.ItemStyle{
							Color: "rgba(238, 238, 238, 0.7)",
						},
					},
					opts.MarkAreaDataItem{
						YAxis: signal.Bar.Low,
					},
				))
			// endregion

			for i, trade := range signal.Trades {
				//var high opts.MarkPointDataItem
				//if trade.Direction == Long && trade.HighAfterBars != 0 {
				//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("High +%0.2f", trade.High-trade.EntryPrice))
				//}
				//if trade.Direction == Short && trade.HighAfterBars != 0 {
				//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("Low +%0.2f", trade.EntryPrice-trade.High))
				//}
				opt = append(opt, charts.WithMarkPointDataItem(
					lArrow(trade.EntryPrice, trade.EntryTime.Truncate(5*time.Minute), "Entry "+trade.Direction.String()),
					lArrow(trade.ExitPrice, trade.ExitTime.Truncate(5*time.Minute), fmt.Sprintf("Exit %0.2f (%s)", trade.Profit, trade.ExitReason)),
				))

				series := charts.SingleSeries{Name: fmt.Sprintf("Trade %d stop", i),
					Type: types.ChartLine,
					Data: trade.PlotStopLine(data),
				}
				line.MultiSeries = append(line.MultiSeries, series)
			}

			line.SetXAxis(data.ToChartXAxis()).
				AddSeries("", data.ToChartData(), opt...).
				AddSeries("20MA", []opts.KlineData{{
					Name:  "data",
					Value: "calculateMA(20, data)",
				}})

			// generate prev and next links
			_, _ = fmt.Fprintf(w, "<a href='/?d=%s'>prev</a> <a href='/?d=%s'>next</a>", pos.prev, pos.next)

			page := components.NewPage()
			page.Initialization.AssetsHost = "/static/"
			page.Assets.AddCustomizedJSAssets("/static/util.js")
			page.Layout = components.PageFlexLayout
			page.AddCharts(line)
			templates.BaseTpl = baseTpl
			err := page.Render(w)
			if err != nil {
				panic(err)
			}

			td := func(inner any) string {
				return fmt.Sprint("<td>", inner, "</td>")
			}

			var pl float64
			for _, trade := range signal.Trades {
				pl += trade.Profit
			}

			summary := strings.Builder{}
			summary.WriteString("<hr/><table border=1><tr>")
			summary.WriteString(fmt.Sprintln("<tr><th>signal</th>", td(signal.Bar.Timestamp.Format("15:04")), td(signal.Bar.High), td(signal.Bar.Low), "<td></td><td></td></tr>"))
			summary.WriteString("<tr><th></th><th>Time</th><th>Price</th><th>Dir/Result</th><th>Reason</th>")
			for _, trade := range signal.Trades {
				summary.WriteString(fmt.Sprintln("<tr><th>entry</th>", td(trade.EntryTime.Format("15:04")), td(trade.EntryPrice), td(trade.Direction), td(trade.OpenReason), "</tr>"))
				summary.WriteString(fmt.Sprintln("<tr><th>exit</th>", td(trade.ExitTime.Format("15:04")), td(trade.ExitPrice), td(trade.Profit), td(trade.ExitReason), "</tr>"))
			}
			summary.WriteString(fmt.Sprintf("<tr><th></th><th></th><th></th><th></th><th>%0.2f</th><th></th>", pl))
			summary.WriteString("</table>")
			_, _ = w.Write([]byte(summary.String()))

			_ = line.Render(w)

		*/

	}
}
