package exchange

import (
	"errors"
	"io"

	"github.com/mwlazlo/srs/internal"
	"github.com/mwlazlo/srs/internal/models"
	"github.com/mwlazlo/srs/internal/td365"
)

func closeTrade(t *models.Trade, tick *models.Tick, balance float64) (newBalance float64) {
	t.Status = models.Closed
	t.ExitTime = tick.Timestamp
	t.ExitPrice = tick.MidPrice()
	t.PointsProfit = calculatePointsProfit(t, t.ExitPrice)
	t.Profit = calculateRealProfit(t, t.ExitPrice)
	newBalance = internal.Round2(balance + t.Profit)
	t.Balance = newBalance
	return
}

func calculatePointsProfit(t *models.Trade, close float64) float64 {
	switch t.Direction {
	case models.Long:
		return internal.Round2(close - t.EntryPrice)
	case models.Short:
		return internal.Round2(t.EntryPrice - close)
	}
	return 0
}

func calculateRealProfit(t *models.Trade, close float64) float64 {
	return internal.Round2(t.Size * calculatePointsProfit(t, close))
}

type Simulator struct {
	reader       *TickReader // internally generate ticks, like a real exchange
	positions    map[int]*models.Trade
	orders       map[int]*models.Trade
	tradeId      int
	currentTick  *models.Tick
	tradeManager *MarketContext
	balance      float64
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

func (s *Simulator) ExitPosition(id int) *models.Trade {
	trade := s.positions[id]
	s.closePosition(trade, s.currentTick)
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
		s.tradeManager.HandleTick(tick)

		if finalLoop {
			return
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

func (s *Simulator) CreateOrder(symbol models.Symbol, direction models.Direction, size, open, stop, target float64) *models.Trade {
	s.tradeId++
	trade := &models.Trade{
		Id:          s.tradeId,
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

func (s *Simulator) enterPosition(trade *models.Trade, tick *models.Tick) {
	trade.EntryTime = tick.Timestamp
	trade.EntryPrice = tick.DirectionPrice(trade.Direction)
	trade.Status = models.Position
	delete(s.orders, trade.Id)
	s.positions[trade.Id] = trade
	s.tradeManager.PositionOpened(trade)
}

func (s *Simulator) closePosition(trade *models.Trade, tick *models.Tick) {
	s.balance = closeTrade(trade, tick, s.balance)
	delete(s.positions, trade.Id)
	s.tradeManager.PositionClosed(trade)
}

/*func (s *Simulator) CloseAllPositions(bar *Bar) {
	for _, trade := range s.positions {
		trade.Close(tick)
		s.tradeManager.TradeClosed(trade)
		delete(s.positions, trade.Id)
	}
}

func (s *Simulator) CloseAllOrders() {
	for _, trade := range s.orders {
		delete(s.orders, trade.Id)
	}
}
*/

func NewExchangeSimulator(file string, handler *MarketContext) *Simulator {
	reader, err := NewTickReader(file)
	if err != nil {
		panic(err)
	}
	rv := &Simulator{
		reader:       reader,
		tradeManager: handler,
		positions:    make(map[int]*models.Trade),
		orders:       make(map[int]*models.Trade),
		balance:      3_450,
	}
	return rv
}
