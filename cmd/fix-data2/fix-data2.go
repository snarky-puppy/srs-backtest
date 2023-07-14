package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/mwlazlo/srs/internal/pp"
)

type Row struct {
	Timestamp time.Time
	Bid       string
	Ask       string
}

func main() {
	fileGroups, err := groupFilesBySymbol("./data/td/raw")
	if err != nil {
		log.Fatal(err)
	}

	for symbol, files := range fileGroups {
		fmt.Println("Symbol:", symbol)
		if !strings.HasPrefix(symbol, "Germ") {
			continue
		}
		err := process(symbol, files)
		if err != nil {
			panic(err)
		}
	}
}

func process(sym string, files []os.DirEntry) error {

	outfp, err := os.Create(fmt.Sprintf("data/td/%s.csv", sym))
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(outfp)
	writer.Comma = ','

	// write CSV header
	if err := writer.Write([]string{"Timestamp", "Bid", "Ask"}); err != nil {
		return err
	}

	prices := make([]pp.Price, 0)

	for _, f := range files {
		fp, err := os.Open(path.Join("data/td/raw", f.Name()))
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(fp)
		for scanner.Scan() {
			var price pp.Price
			if err := json.Unmarshal(scanner.Bytes(), &price); err != nil {
				log.Println(f.Name())
				log.Println(scanner.Text())
				return fmt.Errorf("failed to unmarshal %w", err)
			}
			prices = append(prices, price)
		}
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Timestamp.Before(prices[j].Timestamp)
	})

	prices = removeDuplicates(prices)

	//taipei, err := time.LoadLocation("Asia/Taipei")
	//if err != nil {
	//	panic(err)
	//}

	for _, price := range prices {
		record := []string{
			price.Timestamp.
				AddDate(0, 0, -112).
				//Add(-(12 * time.Hour)).
				//In(taipei).
				UTC().
				Format(time.RFC3339),
			fmt.Sprintf("%f", price.Bid),
			fmt.Sprintf("%f", price.Ask),
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

func removeDuplicates(prices []pp.Price) []pp.Price {
	if len(prices) == 0 {
		return prices
	}

	result := make([]pp.Price, 1, len(prices))
	result[0] = prices[0]

	for _, price := range prices[1:] {
		lastPrice := result[len(result)-1]
		if price.Timestamp != lastPrice.Timestamp || price.High != lastPrice.High || price.Low != lastPrice.Low {
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
