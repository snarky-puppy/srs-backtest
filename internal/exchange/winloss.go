package exchange

import "fmt"

type winloss struct {
	win   int
	loss  int
	even  int
	total int
}

func (w *winloss) addWin() {
	w.win++
	w.total++
}

func (w *winloss) addLoss() {
	w.loss++
	w.total++
}

func (w *winloss) addEven() {
	w.even++
	w.total++
}

// percentage of wins
func (w *winloss) winPercentage() float64 {
	return float64(w.win) / float64(w.total)
}

// percentage of losses
func (w *winloss) lossPercentage() float64 {
	return float64(w.loss) / float64(w.total)
}

// percentage of even
func (w *winloss) evenPercentage() float64 {
	return float64(w.even) / float64(w.total)
}

func (w *winloss) String() string {
	return fmt.Sprintf("%d/%d/%d", w.win, w.even, w.loss)
}
