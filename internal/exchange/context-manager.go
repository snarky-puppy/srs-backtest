package exchange

import (
	"github.com/mwlazlo/srs/internal/models"
	"github.com/mwlazlo/srs/internal/td365"
)

type EntryScanner interface {
	On5MinBar(history *History, marketContext *MarketContext)
	Symbol() models.Symbol
}

type ContextManager struct {
	contexts map[int]*MarketContext // k=Symbol.Key()
	exchange *td365.Platform
}

func (m *ContextManager) Backfill(symbol models.Symbol, bars models.Series) {
	m.contexts[symbol.Key()].Backfill(bars)
}

func (m *ContextManager) HandleTick(tick *models.Tick) {
	m.contexts[tick.Symbol.Key()].HandleTick(tick)
}

func (m *ContextManager) PositionOpened(trade *models.Trade) {
	m.contexts[trade.Symbol.Key()].PositionOpened(trade)
}

func (m *ContextManager) PositionClosed(trade *models.Trade) {
	m.contexts[trade.Symbol.Key()].PositionClosed(trade)
}

func (m *ContextManager) AddContext(marketConfig models.MarketConfig, scanner ...EntryScanner) {
	m.contexts[marketConfig.Symbol().Key()] = NewMarketContext(marketConfig, scanner...)
}

func (m *ContextManager) SetExchange(platform *td365.Platform) {
	m.exchange = platform
	for _, context := range m.contexts {
		context.exchange = platform
	}
}

func (m *ContextManager) SubscribeAll() {
}

func (m *ContextManager) Initialise() {
	for _, context := range m.contexts {
		m.exchange.BackFill(context.marketConfig)
		m.exchange.Subscribe(context.marketConfig.Symbol())
	}
}

type ContextManagerInput struct {
	Scanners []EntryScanner
	Config   models.MarketConfig
}

func NewContextManager(input ...ContextManagerInput) *ContextManager {
	cm := &ContextManager{
		contexts: make(map[int]*MarketContext),
	}
	for _, i := range input {
		cm.AddContext(i.Config, i.Scanners...)
	}
	return cm
}
