package main

import (
	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/strategy"
)

func main() {
	srs := strategy.SrsEntry{}
	tradeManager := exchange.NewMarketContext(&srs, &srs)
	sim := exchange.NewExchangeSimulator(srs.SimData(), tradeManager)
	tradeManager.SetExchange(sim)

	sim.ProcessTicks()
	tradeManager.PrintReport()
	tradeManager.SaveData("data/reports")
}
