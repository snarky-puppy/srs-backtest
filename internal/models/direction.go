package models

import "fmt"

type Direction int

const (
	Long Direction = iota
	Short
)

func (d Direction) String() string {
	switch d {
	case Long:
		return "long"
	case Short:
		return "short"
	default:
		panic(fmt.Sprintf("invalid direction: %d", d))
	}
}
