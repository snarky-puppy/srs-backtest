package main

import (
	"encoding/csv"
	"os"
	"strings"
	"time"
)

type Row struct {
	Timestamp time.Time
	Open      string
	High      string
	Low       string
	Close     string
}

func main() {
	f, err := os.Open("data/dax-5m.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = 7 // date, time, open, high, low, close, volume
	reader.Comma = ';'

	lines, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	rows := make([]Row, len(lines))
	location, _ := time.LoadLocation("America/Costa_Rica") // Costa Rica is in GMT-6 timezone
	london, _ := time.LoadLocation("Europe/London")

	for i, line := range lines { // Skip header
		timestamp, err := time.ParseInLocation("02/01/2006 15:04:05", strings.Join([]string{line[0], line[1]}, " "), location)
		if err != nil {
			panic(err)
		}

		rows[i] = Row{
			Timestamp: timestamp,
			Open:      line[2],
			High:      line[3],
			Low:       line[4],
			Close:     line[5],
		}
	}

	outfile, err := os.Create("data/dax-5m-fixed.csv")
	if err != nil {
		panic(err)
	}
	defer outfile.Close()

	writer := csv.NewWriter(outfile)
	defer writer.Flush()

	writer.Write([]string{"timestamp", "open", "high", "low", "close"}) // write header

	for _, row := range rows {
		record := []string{
			row.Timestamp.In(london).Format(time.RFC3339),
			row.Open,
			row.High,
			row.Low,
			row.Close,
		}
		err := writer.Write(record)
		if err != nil {
			panic(err)
		}
	}
}
