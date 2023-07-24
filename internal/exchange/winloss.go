package exchange

import "fmt"

type winloss struct {
	win  int
	loss int
	even int
}

func (w *winloss) addWin() {
	w.win++
}

func (w *winloss) addLoss() {
	w.loss++
}

func (w *winloss) winRate() int {
	return w.win * 100 / (w.win + w.loss)
}

func (w *winloss) addEven() {
	w.even++
}

func (w *winloss) String() string {
	return fmt.Sprintf("%d/%d/%d", w.win, w.even, w.loss)
}
