package models

import "time"

type Exchange interface {
	Subscribe(symbol Symbol)
	CreateOrder(symbol Symbol, direction Direction, size, open, stop, target float64) *Position
	ExitPosition(id int, tick *Tick) *Position
	CancelOrder(id int)
	UpdatePosition(id int, stop, target float64)
	GetBalance() float64
	RequestBackFill(marketID int, location *time.Location)
}
