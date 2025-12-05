package trader

import (
	"nofx/decision"
	"testing"
)

// fakeSnapshotTrader implements Trader with configurable GetPositions result.
type fakeSnapshotTrader struct {
	positions []map[string]interface{}
	err       error
}

func (f *fakeSnapshotTrader) GetBalance() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (f *fakeSnapshotTrader) GetPositions() ([]map[string]interface{}, error) {
	return f.positions, f.err
}
func (f *fakeSnapshotTrader) OpenLong(string, float64, int) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeSnapshotTrader) OpenShort(string, float64, int) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeSnapshotTrader) CloseLong(string, float64) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeSnapshotTrader) CloseShort(string, float64) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeSnapshotTrader) SetLeverage(string, int) error                        { return nil }
func (f *fakeSnapshotTrader) SetMarginMode(string, bool) error                     { return nil }
func (f *fakeSnapshotTrader) GetMarketPrice(string) (float64, error)               { return 0, nil }
func (f *fakeSnapshotTrader) SetStopLoss(string, string, float64, float64) error   { return nil }
func (f *fakeSnapshotTrader) SetTakeProfit(string, string, float64, float64) error { return nil }
func (f *fakeSnapshotTrader) CancelStopLossOrders(string) error                    { return nil }
func (f *fakeSnapshotTrader) CancelTakeProfitOrders(string) error                  { return nil }
func (f *fakeSnapshotTrader) CancelAllOrders(string) error                         { return nil }
func (f *fakeSnapshotTrader) CancelStopOrders(string) error                        { return nil }
func (f *fakeSnapshotTrader) FormatQuantity(string, float64) (string, error)       { return "", nil }
func (f *fakeSnapshotTrader) GetRecentFills(string, int64, int64) ([]map[string]interface{}, error) {
	return nil, nil
}

func TestRefreshPositionSnapshotAfterExecution_UsesLatestPositions(t *testing.T) {
	// 初始 lastPositions 里有持仓（来自执行前快照）
	at := &AutoTrader{
		trader:                &fakeSnapshotTrader{positions: []map[string]interface{}{}}, // 最新持仓为空
		lastPositions:         map[string]decision.PositionInfo{"BTCUSDT_short": {Symbol: "BTCUSDT", Side: "short"}},
		positionFirstSeenTime: map[string]int64{"BTCUSDT_short": 123},
		positionStopLoss:      map[string]float64{"BTCUSDT_short": 91201},
		positionTakeProfit:    map[string]float64{},
	}

	// ctxPositions 模拟执行前的快照（包含旧持仓）
	ctxPositions := []decision.PositionInfo{
		{Symbol: "BTCUSDT", Side: "short"},
	}

	at.refreshPositionSnapshotAfterExecution(ctxPositions)

	if len(at.lastPositions) != 0 {
		t.Fatalf("expected lastPositions to be cleared by latest positions, got %d", len(at.lastPositions))
	}
}

func TestRefreshPositionSnapshotAfterExecution_FallbackOnError(t *testing.T) {
	// GetPositions 出错时，应回退到 ctxPositions（保持持仓，不误清空）
	at := &AutoTrader{
		trader:                &fakeSnapshotTrader{err: assertError{}},
		lastPositions:         map[string]decision.PositionInfo{"BTCUSDT_short": {Symbol: "BTCUSDT", Side: "short"}},
		positionFirstSeenTime: map[string]int64{},
		positionStopLoss:      map[string]float64{},
		positionTakeProfit:    map[string]float64{},
	}

	ctxPositions := []decision.PositionInfo{
		{Symbol: "BTCUSDT", Side: "short"},
	}

	at.refreshPositionSnapshotAfterExecution(ctxPositions)

	pos, ok := at.lastPositions["BTCUSDT_short"]
	if !ok {
		t.Fatalf("expected fallback to keep BTCUSDT_short in lastPositions")
	}
	if pos.Side != "short" {
		t.Fatalf("expected side short, got %s", pos.Side)
	}
}

// assertError is a sentinel error used to simulate trader failures.
type assertError struct{}

func (assertError) Error() string { return "assert error" }
