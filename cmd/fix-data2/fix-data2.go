package main

import (
	"encoding/csv"
	"errors"
	"fmt"
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
	files, err := os.ReadDir("./data/td")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		symbol := file.Name()
		fmt.Println("Symbol:", symbol)
		if file.Type() == os.ModeDir {
			continue
		}
		err := process(symbol)
		if err != nil {
			panic(err)
		}
	}
}

func process(sym string) error {

	// write CSV header
	//if err := writer.Write([]string{"Timestamp", "Bid", "Ask"}); err != nil {
	//	return err
	//}

	taipei, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		panic(err)
	}

	var data []Row

	fp, err := os.Open(path.Join("data/td", sym))
	if err != nil {
		panic(err)
	}
	reader := csv.NewReader(fp)
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
			Timestamp: ts.In(taipei),
			Bid:       fields[1],
			Ask:       fields[2],
		})
	}
	_ = fp.Close()

	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	data = removeDuplicates(data)

	outfp, err := os.Create(fmt.Sprintf("data/td/new/%s", sym))
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(outfp)
	writer.Comma = ','

	for _, price := range data {
		record := []string{
			price.Timestamp.
				//AddDate(0, 0, -112).
				In(taipei).
				Format(time.RFC3339),
			price.Bid,
			price.Ask,
		}
		if err := writer.Write(record); err != nil {
			panic(err)
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		panic(writer.Error())
	}
	_ = outfp.Close()

	return nil
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
