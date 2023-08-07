package models

import "fmt"

type Symbol struct {
	QuoteID  int
	MarketID int
}

func (s Symbol) Key() string {
	return fmt.Sprintf("%d,%d", s.MarketID, s.QuoteID)
}
