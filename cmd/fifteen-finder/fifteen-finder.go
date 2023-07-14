package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/components"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/mwlazlo/srs/internal/pp"
)

const (
	ScanTTL         = time.Hour * 3
	TargetPoints    = 20
	TrailStopPoints = 15
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
	Period  Period // Start of 15 minute bar to use as triggers
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
	strategy.printResult()
	strategy.runUI()
}

// MarketClose returns the time the market closes on day t
func MarketClose(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 17, 25, 0, 0, t.Location())
}

func MarketOpen(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, t.Location())
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
		srsSignal   *pp.Signal
		rangeSignal *pp.Signal
		trades      []*pp.Trade
	)

	closeAll := func(reason string, idx int, bar *pp.Bar) {
		if len(trades) == 0 {
			return
		}
		for _, trade := range trades {
			trade.Close = bar.Close
			trade.CloseAt = bar.Timestamp
			trade.CloseAtBar = bar
			trade.CloseAtIdx = idx
			trade.CloseReason = "end of life"
		}
		trades = nil
	}

	for i, bar := range bars {
		if i == len(bars)-1 {
			break
		}
		if i < 120 { // enough time for the data to settle
			continue
		}

		if bar.Timestamp.Year() < 2020 {
			continue
		}

		var (
			t       = bar.Timestamp
			prevBar = bars[i-1]
		)

		// refactor this
		if t.Before(MarketOpen(t)) || t.After(MarketClose(t)) {
			// don't go over closing time
			closeAll("market close", i, bar)
			srsSignal = nil
			rangeSignal = nil
			continue
		}

		if len(trades) > 0 {
			trades = manageTrades(i, trades, bar, trades[0].Signal)
			if trades != nil {
				continue
			}
			//continue // don't scan immediately after closing a trade
		}

		if bar.Timestamp.Format("2006-01-02T15:04:05") == "2000-06-06T11:00:00" {
			fmt.Println("break")
		}

		// setup srs signal
		if s.IsPeriod(t) {

			last3Bars := bars[i-3 : i]
			if last3Bars[0].Timestamp.Before(MarketOpen(t)) {
				log.Println("skipping signal due to gap in data", last3Bars[0].Timestamp, last3Bars[1].Timestamp, last3Bars[2].Timestamp)
				continue
			}
			signalBar := last3Bars[0].Copy()
			for _, b := range last3Bars[1:] {
				signalBar.Add(b)
			}
			srsSignal = &pp.Signal{
				Bar:         signalBar,
				Idx:         i - 3,
				BarDuration: 15 * time.Minute,
				CanTradeFn: func(signal pp.Signal, direction pp.Direction) bool {
					// we can trade on each direction only twice
					cnt := 0
					for _, trade := range signal.Trades {
						if trade.Direction == direction && !trade.IsAdditional {
							cnt++
							if cnt >= 3 {
								return false
							}
						}
					}
					return true
				},
			}
			s.Signals = append(s.Signals, srsSignal)
			prevBar = signalBar // fake it for the first bar after the signal
		}

		// check srs signal crossed
		if srsSignal != nil {

			// if our signal is good, enter the trade
			// price has to "cross over" the signal line
			high := srsSignal.High()
			low := srsSignal.Low()
			if prevBar.High <= high && bar.High > high && srsSignal.CanTrade(pp.Long) {
				entry := high // stop order was set here
				stopPrice := stop(srsSignal.Bar, pp.Long)
				trade := pp.NewTrade(pp.Long, entry, stopPrice, target(entry, pp.Long), i, bar, srsSignal, "srs crossover")
				srsSignal.Trades = append(srsSignal.Trades, trade)
				trades = append(trades, trade)
				continue

			} else if prevBar.Low >= low && bar.Low < low && srsSignal.CanTrade(pp.Short) {
				entry := low // stop order was set here
				stopPrice := stop(srsSignal.Bar, pp.Short)
				trade := pp.NewTrade(pp.Short, entry, stopPrice, target(entry, pp.Short), i, bar, srsSignal, "srs crossover")
				srsSignal.Trades = append(srsSignal.Trades, trade)
				trades = append(trades, trade)
				continue
			}
		}

		// setup range signal
		if len(trades) == 0 && rangeSignal == nil {
			// detect trading range in last 10 bars
			idx := i - 10
			tRange := bars[idx].Copy()
			if tRange.Timestamp.Before(MarketOpen(t)) {
				continue
			}

			for _, b := range bars[idx+1 : i] {
				tRange.Add(b)
			}

			if tRange.High-tRange.Low <= 20 {
				rangeSignal = &pp.Signal{
					Bar:         tRange,
					Idx:         idx,
					BarDuration: 10 * (5 * time.Minute),
					CanTradeFn: func(s pp.Signal, direction pp.Direction) bool {
						return len(s.Trades) == 0
					},
				}
				s.Signals = append(s.Signals, rangeSignal)
			}
		}

		// check range signal crossed
		if rangeSignal != nil {
			// if our signal is good, enter the trade
			// price has to "cross over" the signal line
			high := rangeSignal.High()
			low := rangeSignal.Low()
			if prevBar.High <= high && bar.High > high && rangeSignal.CanTrade(pp.Long) {
				entry := high // stop order was set here
				stopPrice := stop(rangeSignal.Bar, pp.Long)
				trade := pp.NewTrade(pp.Long, entry, stopPrice, target(entry, pp.Long), i, bar, rangeSignal, "range crossover")
				rangeSignal.Trades = append(rangeSignal.Trades, trade)
				rangeSignal = nil // cannot use again
				trades = append(trades, trade)
				continue

			} else if prevBar.Low >= low && bar.Low < low && rangeSignal.CanTrade(pp.Short) {
				entry := low // stop order was set here
				stopPrice := stop(rangeSignal.Bar, pp.Short)
				trade := pp.NewTrade(pp.Short, entry, stopPrice, target(entry, pp.Short), i, bar, rangeSignal, "range crossover")
				rangeSignal.Trades = append(rangeSignal.Trades, trade)
				rangeSignal = nil // cannot use again
				trades = append(trades, trade)
				continue
			}
		}
	}

	// close final trade
	closeAll("end of data", len(bars)-1, bars[len(bars)-1])
}

// IsPeriod returns true if the given time is the end of the 15 minute interval we are using for signals
func (s *Strategy) IsPeriod(t time.Time) bool {
	if s.Period.Minute == t.Minute() && s.Period.Hour == t.Hour() {
		return true
	}
	return false
}

// plural
func manageTrades(i int, trades []*pp.Trade, bar *pp.Bar, signal *pp.Signal) (rv []*pp.Trade) {
	for _, trade := range trades {
		if !trade.IsAdditional && trade.CanAddToPosition && len(signal.Trades) < 10 {
			newTrade := considerAddingToPosition(i, trade, bar, signal)
			if newTrade != nil {
				rv = append(rv, newTrade)
				signal.Trades = append(signal.Trades, newTrade)
			}
		}
		t := manageTrade(i, trade, bar, signal)
		if t != nil {
			rv = append(rv, t)
		}
		if len(rv) > 100 {
			log.Println("break")
		}
	}

	return
}

func considerAddingToPosition(i int, winner *pp.Trade, bar *pp.Bar, signal *pp.Signal) *pp.Trade {
	winner.CanAddToPosition = false // only 1 chance to add to position

	// before: total profits: 37601
	// after:  total profits: 19967
	//if winner.Loser >= 2.5 {
	//	return nil
	//}

	// we would add an entry at the top of the previous bar high
	// so if this bar doesn't reach that high, we don't add
	// before: 43092
	// after: 55090
	switch winner.Direction {
	case pp.Long:
		if bar.High < bars[i-1].High {
			return nil
		}
	case pp.Short:
		if bar.Low > bars[i-1].Low {
			return nil
		}
	}

	reachedTarget := winner.Target
	half := 15.0 // float64(TargetPoints / 2)
	// adjust stop to target / 2
	switch winner.Direction {
	case pp.Long:
		winner.AdjustStop(winner.Target-half, i, bar)
		winner.Target = winner.Target + half
	case pp.Short:
		winner.AdjustStop(winner.Target+half, i, bar)
		winner.Target = winner.Target - half
	}
	winner.AutoAdjustStop = true

	//switch winner.Direction {
	//case pp.Long:
	//	if bars[i-1].High >= winner.Target {
	//		return nil
	//	}
	//case pp.Short:
	//	if bars[i-1].Low <= winner.Target {
	//		return nil
	//	}
	//}

	newTrade := pp.NewTrade(winner.Direction, reachedTarget, winner.Stop, winner.Target, i, bar, signal, "add to position")
	newTrade.AutoAdjustStop = true
	newTrade.DisableLoserCheck = true
	newTrade.IsAdditional = true
	return newTrade
}

// singular
func manageTrade(i int, trade *pp.Trade, bar *pp.Bar, signal *pp.Signal) *pp.Trade {
	trade.BarCnt++

	closeTrade := func(reason string, price float64) *pp.Trade {
		trade.Close = price
		trade.CloseAt = bar.Timestamp
		trade.CloseAtBar = bar
		trade.CloseAtIdx = i
		trade.CloseReason = trade.TradeCloseMsg(reason)
		return nil
	}

	//if trade.OpenAtIdx == 804735 {
	//	log.Println("break")
	//}

	// region normal stopped out
	switch {
	case trade.Direction == pp.Long && bar.Low <= trade.Stop:
		return closeTrade("stop", trade.Stop-slippage())
	case trade.Direction == pp.Short && bar.High >= trade.Stop:
		return closeTrade("stop", trade.Stop+slippage())
	}
	// endregion

	// region adjust stops
	// w/o 80,50
	// w 75,45

	if trade.AutoAdjustStop {
		switch trade.Direction {
		case pp.Long:
			// trail by 15 pts
			if bar.Low > trade.Stop+TrailStopPoints {
				trade.AdjustStop(trade.Stop+TrailStopPoints, i, bar)
			}
		case pp.Short:
			if bar.High < trade.Stop-TrailStopPoints {
				trade.AdjustStop(trade.Stop-TrailStopPoints, i, bar)
			}
		}
	}
	// endregion

	trade.CheckLoser(bar)

	/*

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

	// region target
	switch trade.Direction {
	case pp.Long:
		if bar.High >= trade.Target {
			trade.CanAddToPosition = true
			return trade
		}
	case pp.Short:
		if bar.Low <= trade.Target {
			trade.CanAddToPosition = true
			return trade
		}
	}
	// endregion target

	return trade
}

func (s *Strategy) runUI() {

	log.Println("opening http://localhost:8081/")
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", s.httpserver(bars))
	_ = http.ListenAndServe("localhost:8081", nil)
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
		var (
			signalIndex int
			pl          float64
		)

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
						YAxis: trade.Open,
						XAxis: trade.OpenAt,
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
		//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("High +%0.2f", trade.High-trade.Open))
		//}
		//if trade.Direction == pp.Short && trade.HighAfterBars != 0 {
		//	high = lArrow(trade.High, trade.HighAt, fmt.Sprintf("Low +%0.2f", trade.Open-trade.High))
		//}
		//opt = append(opt, charts.WithMarkPointDataItem(
		//	high,
		//	lArrow(trade.Open, trade.OpenAt, "Entry "+trade.Direction.String()),
		//	lArrow(trade.Close, trade.CloseAt, fmt.Sprintf("Exit %0.2f (%s)", trade.Profit(), trade.CloseReason)),
		//))

		// track stop

		//}

		dataStartIdx, data := bars.SignalContext(signal)

		line.SetXAxis(data.ToChartXAxis()).
			AddSeries("", data.ToChartData(), opt...)

		for idx, trade := range signal.Trades {
			series := charts.SingleSeries{Name: fmt.Sprintf("Trade %d stop", idx),
				Type: types.ChartLine,
				Data: trade.PlotStopLine(data, dataStartIdx),
			}
			line.MultiSeries = append(line.MultiSeries, series)
			pl += trade.Profit()
		}

		page := components.NewPage()
		page.Initialization.AssetsHost = "/static/"
		page.AddCharts(line)
		err := page.Render(w)
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

		td := func(inner any) string {
			return fmt.Sprint("<td>", inner, "</td>")
		}

		summary := strings.Builder{}
		summary.WriteString("<hr/><table border=1><tr>")
		summary.WriteString(fmt.Sprintln("<tr><th>signal</th>", td(signal.Bar.Timestamp.Format("15:04")), td(signal.Bar.High), td(signal.Bar.Low), "<td></td><td></td></tr>"))
		summary.WriteString("<tr><th></th><th>Index</th><th>Time</th><th>Price</th><th>Dir/Result</th><th>Reason</th>")
		for _, trade := range signal.Trades {
			summary.WriteString(fmt.Sprintln("<tr><th>entry</th>", td(trade.OpenAtIdx), td(trade.OpenAt.Format("15:04")), td(trade.Open), td(trade.Direction), td(trade.Reason), "</tr>"))
			summary.WriteString(fmt.Sprintln("<tr><th>exit</th>", td(trade.CloseAtIdx), td(trade.CloseAt.Format("15:04")), td(trade.Close), td(trade.Profit()), td(trade.CloseReason), "</tr>"))
		}
		summary.WriteString(fmt.Sprintf("<tr><th></th><th></th><th></th><th></th><th>%0.2f</th><th></th>", pl))
		summary.WriteString("</table></pre>")
		_, _ = w.Write([]byte(summary.String()))
	}
}

func (s *Strategy) printResult() {
	// find what precentage of trades hit target
	var (
		targetHits          int
		targetMiss          int
		noTrades            int
		biggestProfit       *pp.Trade
		biggestLoss         *pp.Trade
		biggestProfitSigIdx int
		biggestLossSigIdx   int
	)
	for sigIdx, signal := range s.Signals {
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
	)
	for _, signal := range s.Signals {
		k := signal.Bar.Timestamp.Weekday().String()
		if intervals[k] == nil {
			intervals[k] = &winloss{}
		}
		for _, trade := range signal.Trades {
			if trade.Profit() > 0 {
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
