package main

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"time"
)

type Row struct {
	Timestamp time.Time
	Bid       string
	Ask       string
}

func main() {
	files, err := os.ReadDir("data/td")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		checkFile(path.Join("data/td", file.Name()))
	}
}

func checkFile(file string) {

	fp, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	reader := csv.NewReader(fp)

	data := make([]Row, 0)

	for {
		fields, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		if fields[0] == "Timestamp" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, fields[0])
		if err != nil {
			panic(err)
		}
		data = append(data, Row{
			Timestamp: ts,
			Bid:       fields[1],
			Ask:       fields[2],
		})
	}

	origCount := len(data)

	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	data = removeDuplicates(data)

	newCount := len(data)

	if origCount != newCount {
		println(file, origCount, newCount)
	}

}

func removeDuplicates(prices []Row) []Row {
	if len(prices) == 0 {
		return prices
	}

	result := make([]Row, 1, len(prices))
	result[0] = prices[0]

	for _, price := range prices[1:] {
		lastPrice := result[len(result)-1]
		if price.Timestamp != lastPrice.Timestamp || price.Bid != lastPrice.Bid || price.Ask != lastPrice.Ask {
			result = append(result, price)
		}
	}

	return result
}
