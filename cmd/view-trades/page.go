package main

import (
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

	tpl := template.Must(template.ParseFiles("cmd/view-trades/page.tpl"))
	if err := tpl.ExecuteTemplate(w, "page.tpl", data); err != nil {
		panic(err)
	}
}
