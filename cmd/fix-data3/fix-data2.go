package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

const LatestDirBase = "./data/latest"

type Row struct {
	Timestamp time.Time
	Bid       string
	Ask       string
}

func main() {
	fileGroups, err := groupFilesBySymbol(path.Join(LatestDirBase, "peak-profits"))
	if err != nil {
		log.Fatal(err)
	}

	for symbol, files := range fileGroups {
		fmt.Println("Symbol:", symbol)
		err := process(symbol, files)
		if err != nil {
			panic(err)
		}
	}
}

func process(sym string, files []os.DirEntry) error {

	outfp, err := os.Create(path.Join(LatestDirBase, fmt.Sprintf("%s.csv", sym)))
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(outfp)
	writer.Comma = ','

	// write CSV header
	//if err := writer.Write([]string{"Timestamp", "Bid", "Ask"}); err != nil {
	//	return err
	//}

	taipei, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		panic(err)
	}

	var data []Row

	for _, f := range files {
		fp, err := os.Open(path.Join(LatestDirBase, "peak-profits", f.Name()))
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
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	data = removeDuplicates(data)

	for _, price := range data {
		record := []string{
			price.Timestamp.Format(time.RFC3339),
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

func groupFilesBySymbol(dir string) (map[string][]os.DirEntry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fileGroups := make(map[string][]os.DirEntry)
	for _, file := range files {
		symbol, err := getSymbolFromFileName(file.Name())
		if err != nil {
			return nil, err
		}

		fileGroups[symbol] = append(fileGroups[symbol], file)
	}

	return fileGroups, nil
}

func getSymbolFromFileName(fileName string) (string, error) {
	components := strings.Split(fileName, "_")
	if len(components) < 3 {
		return "", fmt.Errorf("invalid file name: %s", fileName)
	}

	return components[0], nil
}
