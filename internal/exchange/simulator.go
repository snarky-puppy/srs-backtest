package exchange

import (
	"errors"
	"io"
	"time"
)

type ExTradeStatus int

const (
	Order ExTradeStatus = iota
	Position
	Closed
)

// exchange idea of trade is much simpler than the strategy's idea of a trade
type ExTrade struct {
	Id          int
	Status      ExTradeStatus
	Direction   Direction
	OpenTime    time.Time
	OpenPrice   float64
	EntryTime   time.Time
	EntryPrice  float64
	ExitTime    time.Time
	ExitPrice   float64
	StopPrice   float64
	TargetPrice float64
	Balance     float64
	Profit      float64
}

func (t *ExTrade) close(tick *Tick, balance float64) (newBalance float64) {
	t.Status = Closed
	t.ExitTime = tick.Timestamp
	t.ExitPrice = tick.MidPrice()
	switch t.Direction {
	case Long:
		t.Profit = t.ExitPrice - t.EntryPrice
	case Short:
		t.Profit = t.EntryPrice - t.ExitPrice
	}
	newBalance = balance + t.Profit
	t.Balance = newBalance
	return
}

type handler interface {
	PositionClosed(trade *ExTrade)
	PositionOpened(trade *ExTrade)
	HandleTick(tick *Tick)
}

type Simulator struct {
	reader      *TickReader // internally generate ticks, like a real exchange
	positions   map[int]*ExTrade
	orders      map[int]*ExTrade
	tradeId     int
	currentTick *Tick
	handler     handler
	balance     float64
}

func (s *Simulator) ExitPosition(id int) *ExTrade {
	trade := s.positions[id]
	s.closePosition(trade, s.currentTick)
	delete(s.positions, id)
	return trade
}

func (s *Simulator) CancelOrder(id int) {
	delete(s.orders, id)
}

func (s *Simulator) ProcessTicks() {
	for {
		finalLoop := false
		tick, err := s.reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				finalLoop = true
			} else {
				panic(err)
			}
		}
		if tick != nil {
			s.addTick(tick)
		}
		// let the strategy handle the final tick as nil
		s.handler.HandleTick(tick)

		if finalLoop {
			return
		}
	}
}

func (s *Simulator) addTick(tick *Tick) {

	s.currentTick = tick

	// check for stop loss or take profit
	for _, trade := range s.positions {
		currentPrice := tick.MidPrice()
		switch trade.Direction {
		case Long:
			if trade.StopPrice != 0 && currentPrice <= trade.StopPrice {
				s.closePosition(trade, tick)
			} else if trade.TargetPrice != 0 && currentPrice >= trade.TargetPrice {
				s.closePosition(trade, tick)
			}
		case Short:
			if trade.StopPrice != 0 && currentPrice >= trade.StopPrice {
				s.closePosition(trade, tick)
			} else if trade.TargetPrice != 0 && currentPrice <= trade.TargetPrice {
				s.closePosition(trade, tick)
			}
		}
	}

	// check if orders should be filled
	for _, trade := range s.orders {
		switch trade.Direction {
		case Long:
			if tick.MidPrice() >= trade.TargetPrice {
				s.enterPosition(trade, tick)
			}
		case Short:
			if tick.MidPrice() <= trade.TargetPrice {
				s.enterPosition(trade, tick)
			}
		}
	}
}

func (s *Simulator) CreateOrder(direction Direction, open, stop, target float64) *ExTrade {
	s.tradeId++
	trade := &ExTrade{
		Id:          s.tradeId,
		Status:      Order,
		Direction:   direction,
		OpenPrice:   open,
		OpenTime:    s.currentTick.Timestamp,
		StopPrice:   stop,
		TargetPrice: target,
	}
	s.orders[trade.Id] = trade
	return trade
}

func (s *Simulator) enterPosition(trade *ExTrade, tick *Tick) {
	trade.EntryTime = tick.Timestamp
	trade.EntryPrice = tick.MidPrice()
	trade.Status = Position
	delete(s.orders, trade.Id)
	s.positions[trade.Id] = trade
	s.handler.PositionOpened(trade)
}

func (s *Simulator) closePosition(trade *ExTrade, tick *Tick) {
	s.balance = trade.close(tick, s.balance)
	delete(s.positions, trade.Id)
	s.handler.PositionClosed(trade)
}

/*func (s *Simulator) CloseAllPositions(bar *Bar) {
	for _, trade := range s.positions {
		trade.Close(tick)
		s.handler.TradeClosed(trade)
		delete(s.positions, trade.Id)
	}
}

func (s *Simulator) CloseAllOrders() {
	for _, trade := range s.orders {
		delete(s.orders, trade.Id)
	}
}
*/

func NewExchangeSimulator(file string, handler handler) *Simulator {
	reader, err := NewTickReader(file)
	if err != nil {
		panic(err)
	}
	rv := &Simulator{
		reader:    reader,
		handler:   handler,
		positions: make(map[int]*ExTrade),
		orders:    make(map[int]*ExTrade),
		balance:   10_000,
	}
	return rv
}
