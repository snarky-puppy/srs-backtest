package models

type Symbol struct {
	QuoteID  int
	MarketID int
}

func (s Symbol) Key() int {
	return s.QuoteID
}
