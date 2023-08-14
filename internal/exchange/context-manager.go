package exchange

import (
	"github.com/mwlazlo/srs/internal/models"
)

type ContextManager struct {
	strategies map[int]models.Strategy // k=Symbol.MarketID
	exchange   models.Exchange
}

func (m *ContextManager) Backfill(marketID int, bars models.Series) {
	m.strategies[marketID].Backfill(bars)
}

func (m *ContextManager) HandleTick(symbol models.Symbol, tick *models.Tick) {
	m.strategies[symbol.MarketID].OnTick(tick)
}

func (m *ContextManager) PositionOpened(trade *models.Position) {
	m.strategies[trade.Symbol.MarketID].PositionOpened(trade)
}

func (m *ContextManager) PositionClosed(trade *models.Position) {
	m.strategies[trade.Symbol.MarketID].PositionClosed(trade)
}

func (m *ContextManager) SetExchange(e models.Exchange) {
	m.exchange = e
	for _, context := range m.strategies {
		context.SetExchange(e)
	}
}

func (m *ContextManager) SubscribeAll() {
}

func (m *ContextManager) Initialise() {
	for _, context := range m.strategies {
		m.exchange.RequestBackFill(context.Symbol().MarketID, context.Location())
		m.exchange.Subscribe(context.Symbol())
	}
}

func (m *ContextManager) GetSymbols() (rv []models.Symbol) {
	for _, context := range m.strategies {
		rv = append(rv, context.Symbol())
	}
	return
}

func (m *ContextManager) PrintReport() {
	for _, context := range m.strategies {
		context.PrintReport()
	}
}

func (m *ContextManager) SaveData(dir string) {
	for _, context := range m.strategies {
		context.SaveData(dir, context.Location())
	}
}

func NewContextManager(strategies ...models.Strategy) *ContextManager {
	cm := &ContextManager{
		strategies: make(map[int]models.Strategy),
	}
	for _, s := range strategies {
		cm.strategies[s.Symbol().MarketID] = s
	}
	return cm
}
