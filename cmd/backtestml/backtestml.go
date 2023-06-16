package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-gota/gota/dataframe"
)

func main() {
	irisCsv, err := os.Open("./data/DAX.csv")
	if err != nil {
		log.Fatal(err)
	}

	df := dataframe.ReadCSV(irisCsv)
	fmt.Println(df)
	head := df.Subset([]int{0, 3})
	fmt.Println(head)
}
