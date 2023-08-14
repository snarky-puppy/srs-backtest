package exchange

import (
	"errors"
	"io"
	"time"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
	"github.com/mwlazlo/srs/internal/td365"
)

type Simulator struct {
	positions   map[int]*models.Position
	orders      map[int]*models.Position
	tradeId     int
	currentTick *models.Tick
	ctxMngr     *ContextManager
	balance     float64
	readers     map[int]*TickReader
}

func (s *Simulator) Subscribe(symbol models.Symbol) {
	//TODO implement me
	panic("implement me")
}

func (s *Simulator) RequestBackFill(marketID int, location *time.Location) {
	//TODO implement me
	panic("implement me")
}

func (s *Simulator) GetPopularMarkets() td365.MarketGroup {
	return td365.MarketGroup{}
}

func (s *Simulator) GetBalance() float64 {
	return s.balance
}

func (s *Simulator) UpdatePosition(id int, stop, target float64) {
	trade := s.positions[id]
	trade.StopPrice = stop
	trade.TargetPrice = target
}

func (s *Simulator) ExitPosition(id int) *models.Position {
	trade := s.positions[id]
	s.closePosition(trade, s.currentTick)
	return trade
}

func (s *Simulator) CancelOrder(id int) {
	delete(s.orders, id)
}

func (s *Simulator) ProcessTicks() {
	for _, r := range s.readers {
		for {
			finalLoop := false
			tick, err := r.Next()
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
			s.ctxMngr.HandleTick(r.symbol, tick)

			if finalLoop {
				break
			}
		}
	}
}

func (s *Simulator) addTick(tick *models.Tick) {

	s.currentTick = tick

	// check for stop loss or take profit
	for _, trade := range s.positions {
		switch trade.Direction {
		case models.Long:
			currentPrice := tick.Buy
			if trade.StopPrice != 0 && currentPrice <= trade.StopPrice {
				s.closePosition(trade, tick)
			} else if trade.TargetPrice != 0 && currentPrice >= trade.TargetPrice {
				s.closePosition(trade, tick)
			}
		case models.Short:
			currentPrice := tick.Sell
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
		case models.Long:
			if tick.Buy >= trade.OpenPrice {
				s.enterPosition(trade, tick)
			}
		case models.Short:
			if tick.Sell <= trade.OpenPrice {
				s.enterPosition(trade, tick)
			}
		}
	}
}

func (s *Simulator) CreateOrder(symbol models.Symbol, direction models.Direction, size, open, stop, target float64) *models.Position {
	s.tradeId++
	trade := &models.Position{
		Id:          s.tradeId,
		Symbol:      symbol,
		Size:        size,
		Status:      models.Order,
		Direction:   direction,
		OpenPrice:   open,
		OpenTime:    s.currentTick.Timestamp,
		StopPrice:   stop,
		TargetPrice: target,
	}
	s.orders[trade.Id] = trade
	return trade
}

func (s *Simulator) enterPosition(trade *models.Position, tick *models.Tick) {
	trade.EntryTime = tick.Timestamp
	trade.EntryPrice = tick.DirectionPrice(trade.Direction)
	trade.Status = models.Active
	delete(s.orders, trade.Id)
	s.positions[trade.Id] = trade
	s.ctxMngr.PositionOpened(trade)
}

func (s *Simulator) closePosition(trade *models.Position, tick *models.Tick) {
	s.balance = closeTrade(trade, tick, s.balance)
	delete(s.positions, trade.Id)
	s.ctxMngr.PositionClosed(trade)
}

func closeTrade(t *models.Position, tick *models.Tick, balance float64) (newBalance float64) {
	t.Status = models.Closed
	t.ExitTime = tick.Timestamp
	t.ExitPrice = tick.MidPrice()
	t.PointsProfit = models.CalculatePointsProfit(t, t.ExitPrice)
	t.Profit = models.CalculateRealProfit(t, t.ExitPrice)
	newBalance = internal.Round2(balance + t.Profit)
	t.Balance = newBalance
	return
}

/*func (s *Simulator) CloseAllPositions(bar *Bar) {
	for _, trade := range s.positions {
		trade.Close(tick)
		s.ctxMngr.TradeClosed(trade)
		delete(s.positions, trade.Id)
	}
}

func (s *Simulator) CloseAllOrders() {
	for _, trade := range s.orders {
		delete(s.orders, trade.Id)
	}
}
*/

func NewExchangeSimulator(handler *ContextManager) *Simulator {
	readers := make(map[int]*TickReader)
	for _, symbol := range handler.GetSymbols() {
		readers[symbol.MarketID] = NewTickReader(symbol)
	}
	rv := &Simulator{
		readers:   readers,
		ctxMngr:   handler,
		positions: make(map[int]*models.Position),
		orders:    make(map[int]*models.Position),
		balance:   3_450,
	}
	handler.SetExchange(rv)
	return rv
}
