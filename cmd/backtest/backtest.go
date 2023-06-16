package main

import (
	"github.com/mwlazlo/srs/internal/pp"
)

func main() {
	bars, err := pp.ReadBarsFromCSV("data/dax-5m-fixed.csv", false)
	if err != nil {
		panic(err)
	}

	strategy := pp.NewSRS()
	i := 1000000
	for _, b := range bars {
		i--
		if i <= 0 {
			break
		}
		strategy.NextBar(b)
	}

	strategy.PrintStats()
}
