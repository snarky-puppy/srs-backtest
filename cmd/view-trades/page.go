package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/mwlazlo/srs/internal/exchange"
)

type pageData struct {
	Title  string
	Prev   string
	Next   string
	Candle exchange.Series
}

// render page.tpl
func Render(w http.ResponseWriter, pos position, report exchange.HistoricalRecord) {
	var data = pageData{
		Title:  report.Signal.Bar.Timestamp.Format(time.DateTime),
		Prev:   pos.prev,
		Next:   pos.next,
		Candle: report.Context,
	}
	tpl := template.Must(template.ParseFiles("cmd/view-trades/page.tpl"))
	if err := tpl.ExecuteTemplate(w, "page.tpl", data); err != nil {
		panic(err)
	}
}
