package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
)

type PageTrade struct {
	*models.Trade
	StopLine []string
}

type pageData struct {
	Title  string
	Prev   string
	Next   string
	Series models.Series
	Signal *models.Signal
	Trades []PageTrade
	Profit float64
	Sma5   []string
	Sma25  []string
	Sma50  []string
}

func plotSmaData(data models.Series, smaData []models.SmaHistory) (rv []string) {
	var (
		sIdx int
	)

	// get the first sma that matches the first bar
	for i, bar := range data {
		if smaData[0].Timestamp.Truncate(bar.Duration) == bar.Timestamp {
			sIdx = i
			break
		}
	}

	for _, bar := range data {
		var curSma = "-"
		if smaData[sIdx].Timestamp.Truncate(bar.Duration).Equal(bar.Timestamp) {
			curSma = fmt.Sprintf("%0.4f", smaData[sIdx].Value)
			sIdx++
		}

		rv = append(rv, curSma)
	}

	return
}

// RenderPage page.tpl
func RenderPage(w http.ResponseWriter, pos position, report models.HistoricalRecord) {
	report.Localise()
	var data = pageData{
		Title:  report.Signal.Bar.Timestamp.Format(time.DateTime),
		Prev:   pos.prev,
		Next:   pos.next,
		Series: report.Context,
		Signal: report.Signal,
	}
	data.Profit = 0.0
	for _, trade := range report.Signal.Trades {
		data.Trades = append(data.Trades, PageTrade{
			Trade:    trade,
			StopLine: trade.PlotStopLine(report.Context),
		})
		data.Profit += trade.Profit
	}
	data.Profit = internal.Round2(data.Profit)
	data.Sma5 = plotSmaData(report.Context, report.Sma5)
	data.Sma25 = plotSmaData(report.Context, report.Sma25)
	data.Sma50 = plotSmaData(report.Context, report.Sma50)

	tpl := template.Must(template.ParseFiles("cmd/view-trades/page.tpl"))
	if err := tpl.ExecuteTemplate(w, "page.tpl", data); err != nil {
		panic(err)
	}
}
