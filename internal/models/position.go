package models

import (
	"time"
)

type Position struct {
	Id           int
	Symbol       Symbol
	OpenOrder    Order
	Size         float64
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

type Order struct {
	Id          int
	Symbol      Symbol
	OpenPrice   float64
	StopPrice   float64
	TargetPrice float64
	Direction   Direction
	Size        float64
}
