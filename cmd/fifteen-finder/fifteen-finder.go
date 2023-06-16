package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/mwlazlo/srs/internal/pp"
)

const (
	ScanTTL      = time.Hour * 3
	TargetPoints = 30
)

var (
	bars pp.Series
)

func init() {
	var err error
	bars, err = pp.ReadBarsFromCSV("data/dax-5m-fixed.csv.gz", false)
	if err != nil {
		panic(err)
	}
}

func stop(bar *pp.Bar, d pp.Direction) float64 {
	// win rate 33-36%
	// total profit: 10555
	//return (bar.High + bar.Low) / 2

	// win rate: 41-44%
	// total profit: 12257
	if d == pp.Long {
		return bar.Low
	}
	return bar.High
}

func slippage() float64 {
	return 0
}

func target(entry float64, direction pp.Direction) float64 {
	switch direction {
	case pp.Long:
		return entry + TargetPoints
	case pp.Short:
		return entry - TargetPoints
	default:
		panic("invalid direction")
	}
}

// region structs

type winloss struct {
	win  int
	loss int
	even int
}

func (w *winloss) addWin() {
	w.win++
}

func (w *winloss) addLoss() {
	w.loss++
}

func (w *winloss) winRate() int {
	return w.win * 100 / (w.win + w.loss)
}

func (w *winloss) addEven() {
	w.even++
}

func (w *winloss) String() string {
	return fmt.Sprintf("%d/%d/%d", w.win, w.even, w.loss)
}

type Result struct {
	Period  Period
	WinLoss winloss
	Profit  float64
}

func (r *Result) AddTrade(trade *pp.Trade) {
	r.Profit += trade.Profit()
	switch {
	case trade.Profit() > 0:
		r.WinLoss.addWin()
	case trade.Profit() < 0:
		r.WinLoss.addLoss()
	case trade.Profit() == 0:
		r.WinLoss.addEven()
	}
}

type Period struct {
	Hour   int
	Minute int
}

// endregion

type Strategy struct {
	Period  Period
	Signals []*pp.Signal
}

func main() {

	//opening := Period{Hour: 9, Minute: 30}
	//closing := Period{Hour: 15, Minute: 30}

	//scanTimes()
	normalRun()
}

func scanTimes() {
	results := make([]Result, 0)

	for _, oh := range []int{8, 9, 14, 15} {
		for _, om := range []int{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55} {
			strategy := Strategy{
				Period: Period{Hour: oh, Minute: om},
			}
			strategy.runTrades()
			results = append(results, strategy.GetResult())
		}
	}

	for _, r := range results {
		fmt.Printf("%02d:%02d %d%% (%s) $%d\n", r.Period.Hour, r.Period.Minute, r.WinLoss.winRate(), r.WinLoss.String(), int(r.Profit))
	}
}

func normalRun() {
	period := Period{Hour: 9, Minute: 25}
	strategy := Strategy{
		Period: period,
	}
	strategy.runTrades()
	strategy.runUI()
}

// MarketClose returns the time the market closes on day t
func MarketClose(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

// TradeMaxTime returns the maximum time a trade can be open
func TradeMaxTime(t time.Time) time.Time {
	marketClose := MarketClose(t)
	ttl := t.Add(ScanTTL)
	if ttl.After(marketClose) {
		return marketClose
	}
	return ttl
}

func (s *Strategy) runTrades() {

	var (
		signal     *pp.Signal
		trade      *pp.Trade
		scanTo     time.Time
		isScanning bool
	)

	for i, bar := range bars {
		if i == len(bars)-1 {
			break
		}
		if i < 120 { // enough time for the data to settle
			continue
		}

		var (
			t       = bar.Timestamp
			prevBar = bars[i-1]
		)

		// don't go over closing time
		if isScanning && t.After(scanTo) {
			if trade != nil {
				trade.Close = bars[i-1].Close
				trade.CloseAt = bars[i-1].Timestamp
				trade.CloseAtBar = bars[i-1]
				trade.CloseAtIdx = i - 1
				trade.CloseReason = "end of life"
				trade = nil
			}
			isScanning = false
			continue
		}

		if trade != nil {
			trade = manageTrade(i, trade, bar)
			if trade != nil {
				continue
			}
			continue // don't scan immediately after closing a trade
		}

		if s.IsPeriod(t) {

			last3Bars := bars[i-3 : i]
			if last3Bars[0].Timestamp.Sub(last3Bars[2].Timestamp) > time.Minute*15 {
				log.Println("skipping signal due to gap in data", last3Bars[0].Timestamp, last3Bars[1].Timestamp, last3Bars[2].Timestamp)
				continue
			}
			signalBar := *last3Bars[0]
			for _, b := range last3Bars[1:] {
				if b.High > signalBar.High {
					signalBar.High = b.High
				}
				if b.Low < signalBar.Low {
					signalBar.Low = b.Low
				}
				signalBar.Close = b.Close
			}
			// we are at the decision point, restart the TTL, set up the signal bar
			scanTo = TradeMaxTime(t)
			isScanning = true
			signal = &pp.Signal{Bar: &signalBar, Idx: i - 3}
			s.Signals = append(s.Signals, signal)
			prevBar = &signalBar // fake it for the first bar after the signal
		}

		if signal != nil && len(signal.Trades) >= 2 {
			isScanning = false
		}

		if !isScanning {
			continue
		}

		if !signal.CanTrade() {
			continue
		}

		// if our signal is good, enter the trade
		// price has to "cross over" the signal line
		high := signal.High()
		low := signal.Low()
		if prevBar.High <= high && bar.High > high {
			entry := high // stop order was set here
			stopPrice := stop(signal.Bar, pp.Long)
			stopPoints := math.Abs(entry - stopPrice)
			trade = &pp.Trade{
				Open:       entry,
				Stop:       stopPrice,
				StopPoints: stopPoints,
				StopLog: []*pp.StopLog{{
					Stop:      stopPrice,
					Idx:       i,
					Timestamp: t,
				}},
				Target:    target(entry, pp.Long),
				Direction: pp.Long,
				OpenAtBar: bar,
				OpenAt:    bar.Timestamp,
				High:      bar.High,
			}
			signal.Trades = append(signal.Trades, trade)
		} else if prevBar.Low >= low && bar.Low < low {
			entry := low // stop order was set here
			stopPrice := stop(signal.Bar, pp.Short)
			stopPoints := math.Abs(stopPrice - entry)
			trade = &pp.Trade{
				Open:       entry,
				Stop:       stopPrice,
				StopPoints: stopPoints,
				StopLog: []*pp.StopLog{{
					Stop:      stopPrice,
					Idx:       i,
					Timestamp: t,
				}},
				Target:    target(entry, pp.Short),
				Direction: pp.Short,
				OpenAtBar: bar,
				OpenAt:    bar.Timestamp,
				High:      bar.Low,
			}
			signal.Trades = append(signal.Trades, trade)
		}
	}

	// close last trade
	if trade != nil {
		trade.Close = bars[len(bars)-1].Close
		trade.CloseAt = bars[len(bars)-1].Timestamp
		trade.CloseAtBar = bars[len(bars)-1]
		trade.CloseReason = "end of data"
	}
}

func (s *Strategy) IsPeriod(t time.Time) bool {
	if s.Period.Minute == t.Minute() && s.Period.Hour == t.Hour() {
		return true
	}
	return false
}

func manageTrade(i int, trade *pp.Trade, bar *pp.Bar) *pp.Trade {
	trade.BarCnt++

	/*
				Ideas for managing the trade

				1. if the trade is in loss for too long, 3-5 bars in, close the trade
					- long: the high is under the trigger for 3 bars -- close
					- short: the low is over the trigger for 3 bars -- close
				or
					(dubbed "the straddle rule")
					- long: high above and low below the trigger for 4 consecutive -- close
					- short: low below and high above the trigger for 4 consecutive -- close
		        or
					(modified straddle)
					- straddle for 3 then move out -- set stop to break even
			    or
				  	- long: straddle for 3 then high below the trigger for 1 -- close
					- short: straddle for 3 then low above the trigger for 1 -- close

				** handled by 1. **
				if the price wanders too far from the trigger, but not to the stop,
				try to close for 0
				2000-06-05 09:10 http://localhost:8081/?signalIndex=3

				if a trade triggers and does well (a screamer), but then trends back to the trigger,
				don't take the next trade if it triggers (trending wrong way)
				2000-06-07 09:10 http://localhost:8081/?signalIndex=5

				trade triggers, make a high, dips and makes a double top/bottom,
				close at the peak (and enter the opposite direction?)
				2000-06-12 09:10 http://localhost:8081/?signalIndex=8

				don't have more than 1 sell/buy in a row
			    BUT sometimes this works, test it, remove the restriction
				and count how many times 2 buys in a row profit, or how many times the 2nd fails, etc
				2000-06-13 09:10 http://localhost:8081/?signalIndex=9

				often, after reaching the target, the price will plateau for a few bars
				perhaps create new signal area (box shape) on that plateau
				2000-06-06 09:10 http://localhost:8081/?signalIndex=4

				price is trending -- add to trade
				how to know if it's a trend or screamer?

	*/

	switch trade.Direction {
	case pp.Long:
		// check high
		if bar.High > trade.High {
			trade.High = bar.High
			trade.HighAfterBars = trade.BarCnt
			trade.HighAt = bar.Timestamp
		}
		// adjust stops
		if bar.High > trade.Open+trade.StopPoints {
			//trade.Stop = bar.High - trade.StopPoints // move stop up
			//trade.StopLog = append(trade.StopLog, &srs.StopLog{
			//	Stop:      trade.Stop,
			//	Idx:       i,
			//	Timestamp: bar.Timestamp,
			//})
		}
		// check reached stops
		if bar.Low <= trade.Stop {
			trade.Close = trade.Stop - slippage()
			trade.CloseAt = bar.Timestamp
			trade.CloseAtBar = bar
			trade.CloseAtIdx = i
			trade.CloseReason = "stop"
			return nil
		}
		// check reached target
		if bar.High >= trade.Target {
			trade.Close = trade.Target - slippage()
			trade.CloseAt = bar.Timestamp
			trade.CloseAtBar = bar
			trade.CloseAtIdx = i
			trade.TargetAtBars = trade.BarCnt
			trade.CloseReason = "target"
			return nil
		}
	case pp.Short:
		// check high (low)
		if bar.Low < trade.High {
			trade.High = bar.Low
			trade.HighAfterBars = trade.BarCnt
			trade.HighAt = bar.Timestamp
		}
		// adjust stops
		if bar.Low < trade.Open-trade.StopPoints {
			//trade.Stop = bar.Low + trade.StopPoints // move stop down
			//trade.StopLog = append(trade.StopLog, &srs.StopLog{
			//	Stop:      trade.Stop,
			//	Idx:       i,
			//	Timestamp: bar.Timestamp,
			//})
		}
		// check stops
		if bar.High >= trade.Stop {
			trade.Close = trade.Stop + slippage()
			trade.CloseAt = bar.Timestamp
			trade.CloseAtBar = bar
			trade.CloseAtIdx = i
			trade.CloseReason = "stop"
			return nil
		}
		// check target
		if bar.Low <= trade.Target {
			trade.Close = trade.Target + slippage()
			trade.CloseAt = bar.Timestamp
			trade.CloseAtBar = bar
			trade.CloseAtIdx = i
			trade.TargetAtBars = trade.BarCnt
			trade.CloseReason = "target"
			return nil
		}
	default:
		panic("invalid direction")
	}
	return trade
}

func (s *Strategy) runUI() {

	// print first trade
	//for i := len(bars) - 1; i > 0; i-- {
	//	bar := bars[i]
	//	if bar.Trade != nil {
	//		t := *bar.Trade
	//		t.Signal.Bar.Trade = nil
	//		t.CloseAtBar = nil
	//		t.OpenAtBar = nil
	//		fmt.Printf("dbg index: %d profit=%0.2f\n", i, t.Profit())
	//		if err := json.NewEncoder(os.Stderr).Encode(t); err != nil {
	//			panic(err)
	//		}
	//		break
	//	}
	//}

	// find what precentage of trades hit target
	var (
		targetHits int
		targetMiss int
		noTrades   int
	)
	for _, signal := range s.Signals {
		if len(signal.Trades) == 0 {
			noTrades++
			continue
		}
		for _, trade := range signal.Trades {
			if trade.TargetAtBars != 0 {
				//fmt.Printf("hit: %0.2f\n", bar.Trade.Profit())
				targetHits++
			} else {
				//fmt.Printf("miss: %0.2f\n", bar.Trade.Profit())
				targetMiss++
			}
		}
	}
	fmt.Printf("target hits: %d, target miss: %d\n", targetHits, targetMiss)
	// as a percentage of wins
	fmt.Printf("target hits: %f\n", float64(targetHits)/float64(targetHits+targetMiss))
	fmt.Printf("no trades: %d\n", noTrades)

	// find which day of the week had the most wins, on average
	var (
		// daily intervals
		intervals = make(map[string]*winloss)
	)
	for _, signal := range s.Signals {
		k := signal.Bar.Timestamp.Weekday().String()
		if intervals[k] == nil {
			intervals[k] = &winloss{}
		}
		for _, trade := range signal.Trades {
			if trade.TargetAtBars != 0 {
				intervals[k].addWin()
			} else {
				intervals[k].addLoss()
			}
		}
	}

	// order intervals by day of week
	var keys = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, k := range keys {
		if _, ok := intervals[k]; ok {
			fmt.Printf("minute: %s, winRate: %d (%d/%d)\n", k, intervals[k].winRate(), intervals[k].win, intervals[k].loss)
		}
	}

	// total profit & loss
	var (
		totalProfit float64
	)
	for _, signal := range s.Signals {
		for _, trade := range signal.Trades {
			totalProfit += trade.Profit()
		}
	}
	fmt.Printf("total profit: %f\n", totalProfit)

	log.Println("opening http://localhost:8081/")
	http.HandleFunc("/", s.httpserver(bars))
	_ = http.ListenAndServe(":8081", nil)
}

func (s *Strategy) GetResult() (rv Result) {
	rv.Period = s.Period
	for _, signal := range s.Signals {
		for _, trade := range signal.Trades {
			rv.AddTrade(trade)
		}
	}
	return rv
}

func (s *Strategy) httpserver(bars pp.Series) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signalIndex int

		// take ?index= parameter
		if r.URL.Query().Get("signalIndex") != "" {
			var err error
			i, err := strconv.ParseInt(r.URL.Query().Get("signalIndex"), 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			signalIndex = int(i)
		} else {
			signalIndex = 0
		}

		signal := s.Signals[signalIndex]
		if signal == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		line := createLineChart(signal)

		var opt []charts.SeriesOpts

		log.Println("------------------")
		log.Println("signal", signal.Bar.Timestamp.Format("15:04"), signal.Bar.High, signal.Bar.Low)
		for _, trade := range signal.Trades {
			log.Println("entry", trade.OpenAt.Format("15:04"), trade.Open, trade.Direction)
			log.Println("exit", trade.CloseAt.Format("15:04"), trade.Close)
			log.Println("     profit", trade.Profit())
		}
		arrow := func(y float64, x time.Time, label string) opts.MarkPointDataItem {
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
					XAxis: signal.Bar.Timestamp.Add(10 * time.Minute),
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
			var high opts.MarkPointDataItem
			if trade.Direction == pp.Long && trade.HighAfterBars != 0 {
				high = arrow(trade.High, trade.HighAt, fmt.Sprintf("High +%0.2f", trade.High-trade.Open))
			}
			if trade.Direction == pp.Short && trade.HighAfterBars != 0 {
				high = arrow(trade.High, trade.HighAt, fmt.Sprintf("Low +%0.2f", trade.Open-trade.High))
			}
			opt = append(opt, charts.WithMarkPointDataItem(
				high,
				arrow(trade.Open, trade.OpenAt, "Entry "+trade.Direction.String()),
				arrow(trade.Close, trade.CloseAt, fmt.Sprintf("Exit- %0.2f (%s)", trade.Profit(), trade.CloseReason)),
			))

			// track stop

		}

		data := bars.SignalContext(signal)

		line.SetXAxis(data.ToChartXAxis()).
			AddSeries("", data.ToChartData(), opt...)

		for idx, trade := range signal.Trades {
			series := charts.SingleSeries{Name: fmt.Sprintf("Trade %d stop", idx),
				Type: types.ChartLine,
				Data: trade.PlotStopLine(data),
			}
			line.MultiSeries = append(line.MultiSeries, series)
		}

		err := line.Render(w)
		if err != nil {
			panic(err)
		}

		next := signalIndex + 1
		prev := signalIndex - 1

		if prev >= 0 {
			_, _ = w.Write([]byte(fmt.Sprintf("<a href='/?signalIndex=%d'>&lt;&lt;Prev</a>&nbsp;&nbsp;", prev)))
		}
		if next < len(s.Signals) {
			_, _ = w.Write([]byte(fmt.Sprintf("<a href='/?signalIndex=%d'>Next&gt;&gt;</a>", next)))
		}
		_, _ = w.Write([]byte("<br><br><a href='/'>Home</a>"))
	}
}

func createLineChart(signal *pp.Signal) *charts.Kline {
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
	return line
}
