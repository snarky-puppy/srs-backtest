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

type TickReader struct {
	curDir int
	symbol string
	file   *os.File
	reader *csv.Reader
	gzr    *gzip.Reader
}

// NewTickReader creates a new TickReader with the specified CSV file
func NewTickReader(symbol string) *TickReader {
	return &TickReader{symbol: symbol}
}

func (t *TickReader) openNext() error {
	t.curDir++
	file, err := os.Open(path.Join(BaseDir, strconv.Itoa(t.curDir), fmt.Sprintf("%s.csv.gz", t.symbol)))
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
func (tr *TickReader) Next() (*models.Tick, error) {
	if tr.reader == nil {
		if err := tr.openNext(); err != nil {
			return nil, err
		}
	}
	record, err := tr.reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			if err := tr.openNext(); err != nil {
				return nil, err
			}
			return tr.Next()
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

	return &models.Tick{Timestamp: timestamp.UTC(), Buy: buy, Sell: sell}, nil
}

// Close closes the CSV file
func (tr *TickReader) Close() error {
	return tr.file.Close()
}
