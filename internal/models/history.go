package models

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

const (
	ShortDt   = "02/01 15:04:05"
	ShortTime = "15:04:05"
)

type HistoricalRecord struct {
	Signal   *Signal
	Context  Series
	Timezone string
}

/*func (r *HistoricalRecord) Localise() {
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
}*/

type History struct {
	signals []*Signal
	bars    Series
	Sma5    *SMA
	Sma25   *SMA
	Sma50   *SMA
}

func (h *History) AddBar(bar *Bar) {
	h.bars = append(h.bars, bar)
	h.Sma50.AddBar(bar)
	h.Sma25.AddBar(bar)
	h.Sma5.AddBar(bar)
}

func (h *History) GetBar(offset int) *Bar {
	if len(h.bars) == 0 || int(math.Abs(float64(offset))) > len(h.bars) {
		return nil
	}
	return h.bars[h.CurrentIndex()+offset]
}

func (h *History) GetBars(index int) Series {
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

func (h *History) Backfill(bars Series) {

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
