package pp

type TradeCloseReason int

const (
	StopLossHit TradeCloseReason = iota
	ProfitTargetHit
	TradeClosedManually
)

func (r TradeCloseReason) String() string {
	switch r {
	case StopLossHit:
		return "StopLossHit"
	case ProfitTargetHit:
		return "ProfitTargetHit"
	case TradeClosedManually:
		return "TradeClosedManually"
	}
	return "Unknown"
}
