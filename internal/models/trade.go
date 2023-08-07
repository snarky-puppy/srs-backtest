package models

import "time"

type TradeStatus int

const (
	Order TradeStatus = iota
	Position
	Closed
)

type Trade struct {
	Id           int
	Symbol       Symbol
	Size         float64
	Status       TradeStatus
	Direction    Direction
	OpenTime     time.Time
	OpenPrice    float64
	EntryTime    time.Time
	EntryPrice   float64
	ExitTime     time.Time
	ExitPrice    float64
	StopPrice    float64
	TargetPrice  float64
	Balance      float64
	Profit       float64
	PointsProfit float64
}
