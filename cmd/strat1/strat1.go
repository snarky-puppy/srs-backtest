package main

import (
	"log"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/exchange"
)

const DATA = "data/td/Germany 40 - Rolling Cash.csv"

func main() {
	ctx := internal.Graceful()
	strat1 := Strategy1{}
	exch := exchange.NewExchange(ctx, DATA, &strat1)
	exch.Wait()
	log.Println("read", strat1.bars, "bars")
}

type Strategy1 struct {
	bars int
}

func (s *Strategy1) FiveMinBar(bar *exchange.Bar) {
	s.bars++
}
