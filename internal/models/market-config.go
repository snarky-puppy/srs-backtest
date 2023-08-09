package models

import "time"

type MarketConfig interface {
	MarketOpen(t time.Time) time.Time
	MarketClose(t time.Time) time.Time
	IsOpen(t time.Time) bool
	Location() *time.Location
	Symbol() Symbol
}
