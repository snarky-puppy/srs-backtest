package models

import (
	"time"
)

//MarketOpen(t time.Time) time.Time
//MarketClose(t time.Time) time.Time
//IsOpen(t time.Time) bool

type Strategy interface {
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
