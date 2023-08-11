package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/exchange"
)

type DayRow struct {
	Timestamp time.Time
	Profit    float64
}

type dailyData struct {
	Days []DayRow
}

func createDayRows(report []exchange.HistoricalRecord) []DayRow {
	var rows []DayRow
	for _, record := range report {
		var profit float64
		for _, trade := range record.Signal.Trades {
			profit += trade.Profit
		}
		rows = append(rows, DayRow{
			Timestamp: record.Signal.Bar.Timestamp,
			Profit:    internal.Round2(profit),
		})
	}
	return rows
}

// RenderPage daily.tpl
func RenderDaily(w http.ResponseWriter, report []exchange.HistoricalRecord) {

	var data = dailyData{
		Days: createDayRows(report),
	}

	tpl := template.Must(template.ParseFiles("cmd/view-trades/daily.tpl"))
	if err := tpl.ExecuteTemplate(w, "daily.tpl", data); err != nil {
		panic(err)
	}
}
