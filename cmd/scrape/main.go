package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/pp"
)

func main() {

	ctx := internal.GetGracefulCtx()

	db, err := pp.NewFileDB(ctx, "peak-profits")
	if err != nil {
		log.Fatal(err)
	}

	scraper := NewScraper()
	quoteToName := map[int]string{}

	for _, q := range scraper.PopularQuotes {
		scraper.Subscribe(q)
		quoteToName[q.QuoteID] = strings.ReplaceAll(q.MarketName, "/", "")
	}

	ch := scraper.GetPriceChannel()
	for {
		select {
		case <-ctx.Done():
			<-db.DoneCh
			fmt.Println("Exiting")
			return
		case price := <-ch:
			fmt.Println("Received update", price.String())
			db.Write(quoteToName[price.QuoteID], price.Timestamp, price.Bid, price.Ask)
		}
	}
}
