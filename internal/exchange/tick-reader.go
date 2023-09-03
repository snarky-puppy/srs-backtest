package exchange

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

const BaseDir = "data/td"

/*
Germany 40 - Rolling Cash.csv.gz
US Tech 100 - Rolling Cash.csv.gz
Wall Street 30 - Rolling Cash.csv.gz
*/
func symbolToFile(symbol models.Symbol) string {
	switch symbol.MarketID {
	case 17068:
		return "Germany 40 - Rolling Cash"
	case 20190:
		return "US Tech 100 - Rolling Cash"
	case 17322:
		return "Wall Street 30 - Rolling Cash"
	default:
		panic(fmt.Sprintf("unknown symbol %v", symbol))
	}
}

type TickReader struct {
	curDir   int
	fileName string
	file     *os.File
	reader   *csv.Reader
	gzr      *gzip.Reader
	symbol   models.Symbol
}

// NewTickReader creates a new TickReader with the specified CSV file
func NewTickReader(symbol models.Symbol) *TickReader {
	return &TickReader{
		symbol:   symbol,
		fileName: symbolToFile(symbol),
	}
}

func (t *TickReader) openNext() error {
	t.curDir++
	file, err := os.Open(path.Join(BaseDir, strconv.Itoa(t.curDir), fmt.Sprintf("%s.csv.gz", t.fileName)))
	if err != nil {
		fmt.Println("TickReader reached end of series", err)
		return io.EOF
	}
	gzr, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	t.gzr = gzr
	t.reader = csv.NewReader(gzr)
	return nil
}

// Next reads the next Tick from the CSV file
func (t *TickReader) Next() (*models.Tick, error) {
	if t.reader == nil {
		if err := t.openNext(); err != nil {
			return nil, err
		}
	}
	record, err := t.reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			if err := t.openNext(); err != nil {
				return nil, err
			}
			return t.Next()
		}
		return nil, err
	}

	timestamp, err := time.Parse(time.RFC3339, record[0])
	if err != nil {
		return nil, err
	}

	buy, err := strconv.ParseFloat(record[1], 64)
	if err != nil {
		return nil, err
	}

	sell, err := strconv.ParseFloat(record[2], 64)
	if err != nil {
		return nil, err
	}

	return &models.Tick{Symbol: t.symbol, Timestamp: timestamp.UTC(), Bid: buy, Ask: sell}, nil
}

// Close closes the CSV file
func (t *TickReader) Close() error {
	return t.file.Close()
}
