package pp

type History struct {
	signals []*Signal
	bars    []*Bar
}

func (h *History) AddBar(bar *Bar) {
	h.bars = append(h.bars, bar)
}

func (h *History) GetBar(index int) *Bar {
	return h.bars[len(h.bars)-index]
}

func (h *History) GetBars(index int) []*Bar {
	return h.bars[len(h.bars)-index:]
}

func (h *History) AddSignal(signal *Signal) {

}

func NewHistory() *History {
	return &History{}
}
