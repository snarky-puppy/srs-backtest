package pp

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type History struct {
	signals []*Signal
	bars    Series
}

func (h *History) AddBar(bar *Bar) {
	h.bars = append(h.bars, bar)
}

func (h *History) GetBar(offset int) *Bar {
	if len(h.bars) == 0 || int(math.Abs(float64(offset))) > len(h.bars) {
		return nil
	}
	return h.bars[h.CurrentIndex()+offset]
}

func (h *History) GetBars(index int) []*Bar {
	if len(h.bars) == 0 || int(math.Abs(float64(index))) > len(h.bars) {
		return nil
	}
	return h.bars[h.CurrentIndex()-(index-1):]
}

func (h *History) AddSignal(signal *Signal) {
	signal.Idx = h.CurrentIndex()
	h.signals = append(h.signals, signal)
}

func (h *History) PrintReport() {
	// find what precentage of trades hit target
	var (
		targetHits          int
		targetMiss          int
		noTrades            int
		biggestProfit       *Trade
		biggestLoss         *Trade
		biggestProfitSigIdx int
		biggestLossSigIdx   int
	)
	for sigIdx, signal := range h.signals {
		if len(signal.Trades) == 0 {
			noTrades++
			continue
		}
		for _, trade := range signal.Trades {
			if biggestLoss == nil || trade.Profit() < biggestLoss.Profit() {
				biggestLossSigIdx = sigIdx
				biggestLoss = trade
			}
			if biggestProfit == nil || trade.Profit() > biggestProfit.Profit() {
				biggestProfitSigIdx = sigIdx
				biggestProfit = trade
			}
			if trade.Profit() > 0 {
				//fmt.Printf("hit: %0.2f\n", trade.Profit())
				targetHits++
			} else {
				//fmt.Printf("miss: %0.2f\n", trade.Profit())
				targetMiss++
			}
		}
	}
	fmt.Printf("target hits: %d, target miss: %d\n", targetHits, targetMiss)
	// as a percentage of wins
	fmt.Printf("target hits: %f\n", float64(targetHits)/float64(targetHits+targetMiss))
	fmt.Printf("no trades: %d\n", noTrades)
	fmt.Printf("biggest profit: %0.2f\thttp://localhost:8081/?signalIndex=%d\n", biggestProfit.Profit(), biggestProfitSigIdx)
	fmt.Printf("biggest loss: %0.2f\thttp://localhost:8081/?signalIndex=%d\n", biggestLoss.Profit(), biggestLossSigIdx)

	// find which day of the week had the most wins, on average
	var (
		// daily intervals
		intervals = make(map[string]*winloss)
		all       = &winloss{}
	)
	for _, signal := range h.signals {
		k := signal.Bar.Timestamp.Weekday().String()
		if intervals[k] == nil {
			intervals[k] = &winloss{}
		}
		for _, trade := range signal.Trades {
			if trade.Profit() > 0 {
				intervals[k].addWin()
				all.addWin()
			} else {
				intervals[k].addLoss()
				all.addLoss()
			}
		}
	}

	// order intervals by day of week
	var keys = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, k := range keys {
		if _, ok := intervals[k]; ok {
			fmt.Printf("%s win rate: %d (%d/%d)\n", k, intervals[k].winRate(), intervals[k].win, intervals[k].loss)
		}
	}
	fmt.Printf("overall win rate: %d (%d/%d)\n", all.winRate(), all.win, all.loss)

	// total profit & loss
	var (
		totalProfit float64
	)
	for _, signal := range h.signals {
		for _, trade := range signal.Trades {
			totalProfit += trade.Profit()
		}
	}
	fmt.Printf("total profit: %f\n", totalProfit)
}

func (h *History) CurrentIndex() int {
	return len(h.bars) - 1
}

func (h *History) SaveData(path string) {
	index := 0
	lastDay := ""
	for _, signal := range h.signals {
		day := signal.Bar.Timestamp.Format("2006-01-02-15-04-05")
		if day != lastDay {
			index = 0
			lastDay = day
		}
		name := fmt.Sprintf("%s/%s-%d.html", path, day, index)
		h.saveSignalData(signal, name)
	}
}

func (h *History) saveSignalData(signal *Signal, fileName string) {

	line := h.createLineChart(signal)

	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}

	page := components.NewPage()
	//page.Initialization.AssetsHost = "/static/"
	page.AddCharts(line)
	err = page.Render(f)
	if err != nil {
		panic(err)
	}

	td := func(inner any) string {
		return fmt.Sprint("<td>", inner, "</td>")
	}

	var pl float64
	for _, trade := range signal.Trades {
		pl += trade.Profit()
	}

	summary := strings.Builder{}
	summary.WriteString("<hr/><table border=1><tr>")
	summary.WriteString(fmt.Sprintln("<tr><th>signal</th>", td(signal.Bar.Timestamp.Format("15:04")), td(signal.Bar.High), td(signal.Bar.Low), "<td></td><td></td></tr>"))
	summary.WriteString("<tr><th></th><th>Index</th><th>Time</th><th>Price</th><th>Dir/Result</th><th>Reason</th>")
	for _, trade := range signal.Trades {
		summary.WriteString(fmt.Sprintln("<tr><th>entry</th>", td(trade.OpenAtIdx), td(trade.OpenTime.Format("15:04")), td(trade.OpenPrice), td(trade.Direction), td(trade.Reason), "</tr>"))
		summary.WriteString(fmt.Sprintln("<tr><th>exit</th>", td(trade.CloseAtIdx), td(trade.CloseTime.Format("15:04")), td(trade.ClosePrice), td(trade.Profit()), td(trade.CloseReason), "</tr>"))
	}
	summary.WriteString(fmt.Sprintf("<tr><th></th><th></th><th></th><th></th><th>%0.2f</th><th></th>", pl))
	summary.WriteString("</table>")
	_, _ = f.Write([]byte(summary.String()))
	_ = f.Close()
}

func (h *History) createLineChart(signal *Signal) *charts.Kline {
	// create a new line instance
	line := charts.NewKLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
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
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("%s",
				signal.Bar.Timestamp.Format("2006-01-02 15:04"),
			),
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

	//lArrow := func(y float64, x time.Time, label string) opts.MarkPointDataItem {
	//	return opts.MarkPointDataItem{
	//		Value: label,
	//		ItemStyle: &opts.ItemStyle{
	//			Color:       "auto",
	//			BorderColor: "black",
	//		},
	//		Label: &opts.Label{
	//			Show:     true,
	//			Position: "right",
	//		},
	//		Symbol:       "arrow",
	//		SymbolRotate: 90,
	//		SymbolSize:   10,
	//		YAxis:        y,
	//		XAxis:        x,
	//	}
	//}

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

	for _, trade := range signal.Trades {
		// profit section
		opt = append(opt,
			charts.WithMarkAreaDataItem(
				opts.MarkAreaDataItem{
					YAxis: trade.OpenPrice,
					XAxis: trade.OpenTime,
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
		)

	}

	//for _, trade := range signal.Trades {
	//var high opts.MarkPointDataItem
	//if trade.Direction == pp.Long && trade.HighAfterBars != 0 {
	//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("High +%0.2f", trade.High-trade.OpenPrice))
	//}
	//if trade.Direction == pp.Short && trade.HighAfterBars != 0 {
	//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("Low +%0.2f", trade.OpenPrice-trade.High))
	//}
	//opt = append(opt, charts.WithMarkPointDataItem(
	//	high,
	//	lArrow(trade.OpenPrice, trade.OpenTime, "Entry "+trade.Direction.String()),
	//	lArrow(trade.ClosePrice, trade.CloseTime, fmt.Sprintf("Exit %0.2f (%s)", trade.Profit(), trade.CloseReason)),
	//))

	// track stop

	//}

	data := h.bars.SignalContext2(signal)

	line.SetXAxis(data.ToChartXAxis()).
		AddSeries("", data.ToChartData(), opt...)

	//for idx, trade := range signal.Trades {
	//	series := charts.SingleSeries{Name: fmt.Sprintf("Trade %d stop", idx),
	//		Type: types.ChartLine,
	//		Data: trade.PlotStopLine(data, dataStartIdx),
	//	}
	//	line.MultiSeries = append(line.MultiSeries, series)
	//}
	return line
}

func NewHistory() *History {
	return &History{}
}
