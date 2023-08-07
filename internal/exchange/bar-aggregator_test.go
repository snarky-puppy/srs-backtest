package exchange

import (
	"testing"
	"time"

	"github.com/mwlazlo/srs/internal/models"
)

func TestProcessTick(t *testing.T) {
	duration := 5 * time.Minute

	// Initialize BarAggregator
	ba := &BarAggregator{
		duration: duration,
		bar:      nil,
	}

	// Mock ticks
	ticks := []*models.Tick{
		{Timestamp: time.Date(2023, 7, 13, 10, 0, 0, 0, time.UTC), Buy: 100, Sell: 105}, // 102.5
		{Timestamp: time.Date(2023, 7, 13, 10, 3, 0, 0, time.UTC), Buy: 101, Sell: 106}, // 103.5
		{Timestamp: time.Date(2023, 7, 13, 10, 5, 0, 0, time.UTC), Buy: 102, Sell: 107}, // 104.5
		//{Timestamp: time.Date(2023, 7, 13, 10, 6, 0, 0, time.UTC), Buy: 102, Sell: 107},
	}

	var bars []*Bar

	for _, tick := range ticks {
		bar := ba.ProcessTick(tick)
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
