package exchange

import (
	"errors"
	"io"
	"time"

	"github.com/mwlazlo/srs/internal/pp"
)

type ExTradeStatus int

const (
	Order ExTradeStatus = iota
	Position
)

// exchange idea of trade is much simpler than the strategy's idea of a trade
type ExTrade struct {
	Id          int
	Status      ExTradeStatus
	Direction   pp.Direction
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

func (t *ExTrade) close(tick *pp.Tick, balance float64) (newBalance float64) {
	t.ExitTime = tick.Timestamp
	t.ExitPrice = tick.MidPrice()
	switch t.Direction {
	case pp.Long:
		t.Profit = t.ExitPrice - t.EntryPrice
	case pp.Short:
		t.Profit = t.EntryPrice - t.ExitPrice
	}
	newBalance = balance + t.Profit
	t.Balance = newBalance
	return
}

type handler interface {
	PositionClosed(trade *ExTrade)
	PositionOpened(trade *ExTrade)
	OrderOpened(trade *ExTrade)
	HandleTick(tick *pp.Tick)
}

type Simulator struct {
	reader      *pp.TickReader // internally generate ticks, like a real exchange
	positions   map[int]*ExTrade
	orders      map[int]*ExTrade
	tradeId     int
	currentTick *pp.Tick
	handler     handler
	balance     float64
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

func (s *Simulator) addTick(tick *pp.Tick) {

	s.currentTick = tick

	// check for stop loss or take profit
	for _, trade := range s.positions {
		currentPrice := tick.MidPrice()
		switch trade.Direction {
		case pp.Long:
			if trade.StopPrice != 0 && currentPrice <= trade.StopPrice {
				s.stoppedOut(trade, tick)
			} else if trade.TargetPrice != 0 && currentPrice >= trade.TargetPrice {
				s.takeProfit(trade, tick)
			}
		case pp.Short:
			if trade.StopPrice != 0 && currentPrice >= trade.StopPrice {
				s.stoppedOut(trade, tick)
			} else if trade.TargetPrice != 0 && currentPrice <= trade.TargetPrice {
				s.takeProfit(trade, tick)
			}
		}
	}

	// check if orders should be filled
	for _, trade := range s.orders {
		switch trade.Direction {
		case pp.Long:
			if tick.MidPrice() >= trade.TargetPrice {
				s.createPosition(trade, tick)
			}
		case pp.Short:
			if tick.MidPrice() <= trade.TargetPrice {
				s.createPosition(trade, tick)
			}
		}
	}
}

func (s *Simulator) CreateOrder(trade *ExTrade) *ExTrade {
	s.tradeId++
	trade.Id = s.tradeId
	trade.OpenTime = s.currentTick.Timestamp
	s.orders[trade.Id] = trade
	return trade
}

func (s *Simulator) createPosition(trade *ExTrade, tick *pp.Tick) {
	trade.EntryTime = tick.Timestamp
	delete(s.orders, trade.Id)
	s.positions[trade.Id] = trade
	s.handler.PositionOpened(trade)
}

func (s *Simulator) stoppedOut(trade *ExTrade, tick *pp.Tick) {
	s.balance = trade.close(tick, s.balance)
	delete(s.positions, trade.Id)
	s.handler.PositionClosed(trade)
}

func (s *Simulator) takeProfit(trade *ExTrade, tick *pp.Tick) {
	s.balance = trade.close(tick, s.balance)
	delete(s.positions, trade.Id)
	s.handler.PositionClosed(trade)
}

/*func (s *Simulator) CloseAllPositions(bar *pp.Bar) {
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
	reader, err := pp.NewTickReader(file)
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
