package models

import (
	"time"
)

type Strategy interface {
	MarketOpen(t time.Time) time.Time
	MarketClose(t time.Time) time.Time
	IsOpen(t time.Time) bool
	Location() *time.Location
	Symbol() Symbol
	Backfill(Series)
	OnTick(tick *Tick)
	PositionOpened(position *Position)
	PositionClosed(position *Position)
	SetExchange(e Exchange)
	PrintReport()
	SaveData(dir string, location *time.Location)
}
