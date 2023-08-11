package exchange

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/mwlazlo/srs/internal/models"
)

const (
	ChartBar = time.Minute * 5
)

type HistoricalRecord struct {
	Signal   *Signal
	Context  models.Series
	Timezone string
}

func (r *HistoricalRecord) Localise() {
	location, _ := time.LoadLocation(r.Timezone)
	for _, bar := range r.Context {
		bar.Timestamp = bar.Timestamp.In(location)
	}
	r.Signal.Bar.Timestamp = r.Signal.Bar.Timestamp.In(location)
	for _, trade := range r.Signal.Trades {
		trade.OpenTime = trade.OpenTime.In(location)
		trade.EntryTime = trade.EntryTime.In(location)
		trade.ExitTime = trade.ExitTime.In(location)
		for _, stopLog := range trade.StopLog {
			stopLog.Timestamp = stopLog.Timestamp.In(location)
		}
		// round profit 2 decimals
		trade.Profit = math.Round(trade.Profit*100) / 100
	}
}

type History struct {
	signals []*Signal
	bars    models.Series
	Sma5    *SMA
	Sma25   *SMA
	Sma50   *SMA
}

func (h *History) AddBar(bar *models.Bar) {
	h.bars = append(h.bars, bar)
	h.Sma50.AddBar(bar)
	h.Sma25.AddBar(bar)
	h.Sma5.AddBar(bar)
}

func (h *History) GetBar(offset int) *models.Bar {
	if len(h.bars) == 0 || int(math.Abs(float64(offset))) > len(h.bars) {
		return nil
	}
	return h.bars[h.CurrentIndex()+offset]
}

func (h *History) GetBars(index int) models.Series {
	if index < 0 {
		panic("index must be positive")
	}
	if len(h.bars) == 0 || int(math.Abs(float64(index))) > len(h.bars) {
		return nil
	}
	return h.bars[h.CurrentIndex()-(index-1):]
}

func (h *History) AddSignal(signal *Signal) {
	h.signals = append(h.signals, signal)
}

func (h *History) PrintReport() {
	var (
		noTrades      int
		biggestProfit *Trade
		biggestLoss   *Trade
		intervals     = make(map[string]*winloss)
		all           = &winloss{}
		totalProfit   float64
	)

	for _, signal := range h.signals {
		if len(signal.Trades) == 0 {
			noTrades++
			continue
		}
		k := signal.Bar.Timestamp.Weekday().String()
		if intervals[k] == nil {
			intervals[k] = &winloss{}
		}
		for _, trade := range signal.Trades {
			fmt.Printf("%s\t%s\t%0.2f\t%s\t%0.2f\t%0.2f\t%0.2f\n",
				trade.Direction,
				trade.EntryTime.Format(ShortDt),
				trade.EntryPrice,
				trade.ExitTime.Format(ShortTime), trade.ExitPrice,
				trade.Profit,
				trade.Balance)

			if trade.Profit == 0 {
				intervals[k].addEven()
				all.addEven()
			} else if trade.Profit > 0 {
				intervals[k].addWin()
				all.addWin()
			} else {
				intervals[k].addLoss()
				all.addLoss()
			}

			if biggestLoss == nil || trade.Profit < biggestLoss.Profit {
				biggestLoss = trade
			}
			if biggestProfit == nil || trade.Profit > biggestProfit.Profit {
				biggestProfit = trade
			}

			totalProfit += trade.Profit
		}
	}

	fmt.Printf("target profits: %d, break even: %d, loss: %d\n", all.win, all.even, all.loss)
	// as a percentage of wins
	fmt.Printf("percentage wins: %0.2f\n", all.winPercentage())
	fmt.Printf("percentage losses: %0.2f\n", all.lossPercentage())
	fmt.Printf("percentage even: %0.2f\n", all.evenPercentage())
	fmt.Printf("no trade signals: %d\n", noTrades)
	fmt.Printf("biggest profit: %0.2f\thttp://localhost:8081/?d=%s\n", biggestProfit.Profit, biggestProfit.Signal.Bar.Timestamp.Format("20060102-150405-0.json"))
	fmt.Printf("biggest loss: %0.2f\thttp://localhost:8081/?d=%s\n", biggestLoss.Profit, biggestLoss.Signal.Bar.Timestamp.Format("20060102-150405-0.json"))

	// order intervals by day of week
	var keys = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, k := range keys {
		if _, ok := intervals[k]; ok {
			fmt.Printf("%s win rate: %0.2f (%s)\n", k, intervals[k].winPercentage(), intervals[k])
		}
	}
	fmt.Printf("overall win rate: %0.2f (%s)\n", all.winPercentage(), all)
	fmt.Printf("total profit: %f\n", totalProfit)
}

func (h *History) CurrentIndex() int {
	return len(h.bars) - 1
}

func (h *History) SaveData(path string, location *time.Location) {
	index := 0
	lastDay := ""
	for _, signal := range h.signals {
		day := signal.Bar.Timestamp.Local().Format("20060102-150405")
		if day != lastDay {
			index = 0
			lastDay = day
		}
		name := fmt.Sprintf("%s/%s-%d.json", path, day, index)
		h.saveSignalData(signal, name, location)
	}
}

func (h *History) saveSignalData(signal *Signal, fileName string, location *time.Location) {
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	body := HistoricalRecord{
		Signal:   signal.EncodeableClone(),
		Context:  h.bars.FilterDay(signal.Bar.Timestamp, location),
		Timezone: location.String(),
	}

	err = enc.Encode(body)
	if err != nil {
		panic(err)
	}
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

	for _, trade := range signal.Trades {
		//var high opts.MarkPointDataItem
		//if trade.Direction == Long && trade.HighAfterBars != 0 {
		//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("High +%0.2f", trade.High-trade.OpenPrice))
		//}
		//if trade.Direction == Short && trade.HighAfterBars != 0 {
		//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("Low +%0.2f", trade.OpenPrice-trade.High))
		//}
		opt = append(opt, charts.WithMarkPointDataItem(
			lArrow(trade.OpenPrice, trade.OpenTime, "Entry "+trade.Direction.String()),
			lArrow(trade.ExitPrice, trade.ExitTime.Truncate(5*time.Minute), fmt.Sprintf("Exit %0.2f (%s)", trade.Profit, trade.ExitReason)),
		))
	}

	// track stop

	//}

	// region data context
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
	startIdx := sort.Search(len(h.bars), func(i int) bool {
		return h.bars[i].Timestamp.After(start)
	})
	endIdx := sort.Search(len(h.bars), func(i int) bool {
		return h.bars[i].Timestamp.After(end)
	})
	data := h.bars[startIdx:endIdx]
	// endregion data context

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

func (h *History) FindAverageLow(n int) float64 {
	var sum float64
	bars := len(h.bars)
	for i := bars - n; i < bars; i++ {
		sum += h.bars[i].Low
	}
	return sum / float64(n)
}

func (h *History) FindAverageHigh(n int) float64 {
	var sum float64
	bars := len(h.bars)
	for i := bars - n; i < bars; i++ {
		sum += h.bars[i].High
	}
	return sum / float64(n)
}

func (h *History) Backfill(bars models.Series) {

}

func NewHistory() *History {
	return &History{
		Sma50: &SMA{
			Period: 50,
		},
		Sma25: &SMA{
			Period: 25,
		},
		Sma5: &SMA{
			Period: 5,
		},
	}
}
