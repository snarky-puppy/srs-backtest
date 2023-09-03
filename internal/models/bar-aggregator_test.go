package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessTick(t *testing.T) {
	duration := 5 * time.Minute

	// Initialize BarAggregator
	ba := &BarAggregator{
		duration: duration,
		bar:      nil,
	}

	// Mock ticks
	ticks := []*Tick{
		{Timestamp: time.Date(2023, 7, 13, 10, 0, 0, 0, time.UTC), Bid: 100, Ask: 105}, // 102.5
		{Timestamp: time.Date(2023, 7, 13, 10, 3, 0, 0, time.UTC), Bid: 101, Ask: 106}, // 103.5
		{Timestamp: time.Date(2023, 7, 13, 10, 5, 0, 0, time.UTC), Bid: 102, Ask: 107}, // 104.5
		//{Timestamp: time.Date(2023, 7, 13, 10, 6, 0, 0, time.UTC), Bid: 102, Ask: 107},
	}

	var bars []*Bar

	for _, tick := range ticks {
		bar := ba.AddTick(tick)
		if bar != nil {
			bars = append(bars, bar)
			t.Logf("bar: %v", bar)
		}
	}

	// We expect two bars, since the third tick should start a new bar
	if len(bars) != 2 {
		t.Errorf("Expected two bars, got %v", len(bars))
	}

	// Check the values of the first bar
	bar := bars[0]
	if bar.Open != ticks[0].MidPrice() {
		t.Errorf("Expected first bar open to be %v, got %v", ticks[0].MidPrice(), bar.Open)
	}
	if bar.Close != ticks[2].MidPrice() {
		t.Errorf("Expected first bar close to be %v, got %v", ticks[2].MidPrice(), bar.Close)
	}
	if !bar.Timestamp.Equal(ticks[0].BaseTime(duration)) {
		t.Errorf("Expected first bar timestamp to be %v, got %v", ticks[0].BaseTime(duration), bar.Timestamp)
	}

	// Check the values of the second bar
	bar = bars[1]
	if bar.Open != ticks[2].MidPrice() {
		t.Errorf("Expected second bar open to be %v, got %v", ticks[2].MidPrice(), bar.Open)
	}
	if !bar.Timestamp.Equal(ticks[2].BaseTime(duration)) {
		t.Errorf("Expected second bar timestamp to be %v, got %v", ticks[2].BaseTime(duration), bar.Timestamp)
	}
}

func TestBarAggregator_AddBar(t *testing.T) {

	// mock 1 minute bars
	bars := []*Bar{
		{Timestamp: time.Date(2023, 7, 13, 10, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 95, Close: 102.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 1, 0, 0, time.UTC), Open: 101, High: 106, Low: 96, Close: 103.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 2, 0, 0, time.UTC), Open: 102, High: 107, Low: 97, Close: 104.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 3, 0, 0, time.UTC), Open: 103, High: 108, Low: 98, Close: 105.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 4, 0, 0, time.UTC), Open: 104, High: 109, Low: 99, Close: 106.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 5, 0, 0, time.UTC), Open: 105, High: 110, Low: 100, Close: 107.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 6, 0, 0, time.UTC), Open: 106, High: 111, Low: 101, Close: 108.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 7, 0, 0, time.UTC), Open: 107, High: 112, Low: 102, Close: 109.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 8, 0, 0, time.UTC), Open: 108, High: 113, Low: 103, Close: 110.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 9, 0, 0, time.UTC), Open: 109, High: 114, Low: 104, Close: 111.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 10, 0, 0, time.UTC), Open: 110, High: 115, Low: 105, Close: 112.5},
		{Timestamp: time.Date(2023, 7, 13, 10, 11, 0, 0, time.UTC), Open: 111, High: 116, Low: 106, Close: 113.5},
	}

	ba := &BarAggregator{
		duration: 5 * time.Minute,
		bar:      nil,
	}
	r := require.New(t)

	r.Nil(ba.AddBar(bars[0]))
	r.Nil(ba.AddBar(bars[1]))
	r.Nil(ba.AddBar(bars[2]))
	r.Nil(ba.AddBar(bars[3]))
	r.Nil(ba.AddBar(bars[4]))
	b := ba.AddBar(bars[5])
	r.NotNil(b)
	r.Equal(bars[0].Timestamp, b.Timestamp)
	r.Equal(bars[0].Open, b.Open)
	r.Equal(bars[4].Close, b.Close)
	r.Equal(bars[4].High, b.High)
	r.Equal(bars[0].Low, b.Low)

	r.Nil(ba.AddBar(bars[6]))
	r.Nil(ba.AddBar(bars[7]))
	r.Nil(ba.AddBar(bars[8]))
	r.Nil(ba.AddBar(bars[9]))
	b = ba.AddBar(bars[10])
	r.NotNil(b)
	r.Equal(bars[5].Timestamp, b.Timestamp)
	r.Equal(bars[5].Open, b.Open)
	r.Equal(bars[9].Close, b.Close)
	r.Equal(bars[9].High, b.High)
	r.Equal(bars[5].Low, b.Low)

	r.Nil(ba.AddBar(bars[11]))
	b = ba.LastBar()
	r.NotNil(b)
	r.Equal(bars[10].Timestamp, b.Timestamp)
	r.Equal(bars[10].Open, b.Open)
	r.Equal(bars[11].Close, b.Close)
	r.Equal(bars[11].High, b.High)
	r.Equal(bars[10].Low, b.Low)
}
