package exchange

import (
	"context"

	"github.com/mwlazlo/srs/internal/pp"
)

// create a new trade when nTicks == 0
// simulates the async-ness of a real exchange
type newTradeSim struct {
	nTicks int
	trade  *pp.Trade
}

type TradeManager struct {
	tickChan     chan *pp.Tick
	openTrades   map[int]*pp.Trade
	closedTrades map[int]*pp.Trade
	strategy     Strategy
	newTradeChan chan *pp.Trade
}

func (m *TradeManager) addTick(tick *pp.Tick) {
	m.tickChan <- tick
}

func (m *TradeManager) run(ctx context.Context) {
	newTrades := []*newTradeSim{}
	tradeId := 0

	for {
		select {
		case <-ctx.Done():
			return
		case newTrade := <-m.newTradeChan:
			tradeId++
			newTrade.Id = tradeId
			newTrades = append(newTrades, &newTradeSim{
				nTicks: 2,
				trade:  newTrade,
			})
		case tick := <-m.tickChan:
			// check for stop loss
			for _, trade := range m.openTrades {
				p := tick.MidPrice()
				switch trade.Direction {
				case pp.Long:
					if p <= trade.StopPrice {
						m.StoppedOut(trade, tick)
					}
				case pp.Short:
					if p >= trade.StopPrice {
						m.StoppedOut(trade, tick)
					}
				}
			}

			// check for new trades
			putBack := []*newTradeSim{}
			for _, newTrade := range newTrades {
				newTrade.nTicks--
				if newTrade.nTicks == 0 {
					trade := newTrade.trade
					trade.OpenTime = tick.Timestamp
					trade.OpenPrice = tick.MidPrice()
					m.openTrades[trade.Id] = trade
					m.strategy.TradeOpened(trade)
				} else {
					putBack = append(putBack, newTrade)
				}
			}
			newTrades = putBack

			// check for new take profit
			// check for closing trades
		}
	}
}

func (m TradeManager) CreateTrade(trade *pp.Trade) {
	m.newTradeChan <- trade
}

func (m *TradeManager) StoppedOut(trade *pp.Trade, tick *pp.Tick) {
	trade.ClosePrice = tick.MidPrice()
	trade.CloseReason = "stop"
	m.closedTrades[trade.Id] = trade
	delete(m.openTrades, trade.Id)
	m.strategy.TradeStoppedOut(trade)
}

func NewTradeManager(ctx context.Context, strategy Strategy) *TradeManager {
	rv := &TradeManager{
		tickChan:     make(chan *pp.Tick),
		openTrades:   make(map[int]*pp.Trade),
		closedTrades: make(map[int]*pp.Trade),
		strategy:     strategy,
	}
	go rv.run(ctx)
	return rv
}
