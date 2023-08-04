package exchange

import "testing"

func TestCrossedOverUnder(t *testing.T) {
	upward := &SMA{
		History: []float64{1, 2, 3, 4, 5},
	}

	downward := &SMA{
		History: []float64{5, 4, 3, 2, 1},
	}

	// Test DownToUp crossover
	if !upward.CrossedOver(downward, 5) {
		t.Errorf("Expected DownToUp crossover, got false")
	}

	// Test UpToDown crossover
	if !downward.CrossedUnder(upward, 5) {
		t.Errorf("Expected UpToDown crossover, got false")
	}

	// Test no crossover
	if upward.CrossedUnder(downward, 5) {
		t.Errorf("Expected no crossover, got true")
	}
	if downward.CrossedOver(upward, 5) {
		t.Errorf("Expected no crossover, got true")
	}
}
