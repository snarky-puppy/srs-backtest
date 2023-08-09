package main

import (
	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/exchange"
	"github.com/mwlazlo/srs/internal/strategy"
	"github.com/mwlazlo/srs/internal/td365"
)

/*
2023/08/07 09:10:56 account 6050515 DEMO 2365530 234.84
2023/08/07 09:10:56 account 208675 DEMO 2257550 6942.39
2023/08/07 09:10:56 account 208672 LIVE 5078788 3394.97
*/
const (
	DemoAccountId = 6050515
	ProdAccountId = 208672
)

func main() {

	ctx := internal.GetGracefulCtx()

	strat1 := strategy.SrsEntry{}
	cm := exchange.NewContextManager(exchange.ContextManagerInput{
		Scanners: []exchange.EntryScanner{&strat1},
		Config:   &strat1,
	})
	platform := td365.LaunchPlatform(ctx, DemoAccountId, cm)
	cm.SetExchange(platform)
	cm.Initialise()

}
