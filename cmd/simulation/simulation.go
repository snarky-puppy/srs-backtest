package main

import (
	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/strategy"
)

func main() {
	mngr := exchange.NewContextManager(strategy.NewDaxAggro())
	sim := exchange.NewExchangeSimulator(mngr)
	sim.ProcessTicks()
	mngr.PrintReport()
	mngr.SaveData("data/reports")
}
