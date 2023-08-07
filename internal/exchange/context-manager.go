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
	contexts map[string]*MarketContext // k=Symbol.Key()
	exchange *td365.Platform
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

func (m *ContextManager) AddContext(scanner EntryScanner) {
	m.contexts[scanner.Symbol().Key()] = NewMarketContext(scanner)
}

func (m *ContextManager) SetExchange(platform *td365.Platform) {
	m.exchange = platform
	for _, context := range m.contexts {
		context.exchange = platform
	}
}

func (m *ContextManager) SubscribeAll() {
	for _, context := range m.contexts {
		m.exchange.(context.Symbol())
	}
}

func NewContextManager(scanners ...EntryScanner) *ContextManager {
	cm := &ContextManager{
		contexts: make(map[string]*MarketContext),
	}
	for _, scanner := range scanners {
		cm.AddContext(scanner)
	}
	return cm
}
