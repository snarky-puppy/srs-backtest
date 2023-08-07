package exchange

import (
	"compress/gzip"
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

type TickReader struct {
	file   *os.File
	reader *csv.Reader
}

// NewTickReader creates a new TickReader with the specified CSV file
func NewTickReader(filename string) (*TickReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	reader := csv.NewReader(gzr)
	return &TickReader{file: file, reader: reader}, nil
}

// Next reads the next Tick from the CSV file
func (tr *TickReader) Next() (*models.Tick, error) {
	record, err := tr.reader.Read()
	if err != nil {
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
