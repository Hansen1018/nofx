package market

import (
	"testing"
)

// generateTestKlines 定义在 indicators_test.go 中

// TestCalculateIntradaySeries_VolumeCollection 测试 Volume 数据收集
func TestCalculateIntradaySeries_VolumeCollection(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		expectedVolLen int
	}{
		{
			name:           "正常情况 - 20个K线",
			klineCount:     20,
			expectedVolLen: 10, // 应该收集最近10个
		},
		{
			name:           "刚好10个K线",
			klineCount:     10,
			expectedVolLen: 10,
		},
		{
			name:           "少于10个K线",
			klineCount:     5,
			expectedVolLen: 5, // 应该返回所有5个
		},
		{
			name:           "超过10个K线",
			klineCount:     30,
			expectedVolLen: 10, // 应该只返回最近10个
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if len(data.Volume) != tt.expectedVolLen {
				t.Errorf("Volume length = %d, want %d", len(data.Volume), tt.expectedVolLen)
			}

			// 验证 Volume 数据正确性
			if len(data.Volume) > 0 {
				// 计算期望的起始索引
				start := tt.klineCount - 10
				if start < 0 {
					start = 0
				}

				// 验证第一个 Volume 值
				expectedFirstVolume := klines[start].Volume
				if data.Volume[0] != expectedFirstVolume {
					t.Errorf("First volume = %.2f, want %.2f", data.Volume[0], expectedFirstVolume)
				}

				// 验证最后一个 Volume 值
				expectedLastVolume := klines[tt.klineCount-1].Volume
				lastVolume := data.Volume[len(data.Volume)-1]
				if lastVolume != expectedLastVolume {
					t.Errorf("Last volume = %.2f, want %.2f", lastVolume, expectedLastVolume)
				}
			}
		})
	}
}

// TestCalculateIntradaySeries_VolumeValues 测试 Volume 值的正确性
func TestCalculateIntradaySeries_VolumeValues(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, High: 101.0, Low: 99.0, Open: 100.0},
		{Close: 101.0, Volume: 1100.0, High: 102.0, Low: 100.0, Open: 101.0},
		{Close: 102.0, Volume: 1200.0, High: 103.0, Low: 101.0, Open: 102.0},
		{Close: 103.0, Volume: 1300.0, High: 104.0, Low: 102.0, Open: 103.0},
		{Close: 104.0, Volume: 1400.0, High: 105.0, Low: 103.0, Open: 104.0},
		{Close: 105.0, Volume: 1500.0, High: 106.0, Low: 104.0, Open: 105.0},
		{Close: 106.0, Volume: 1600.0, High: 107.0, Low: 105.0, Open: 106.0},
		{Close: 107.0, Volume: 1700.0, High: 108.0, Low: 106.0, Open: 107.0},
		{Close: 108.0, Volume: 1800.0, High: 109.0, Low: 107.0, Open: 108.0},
		{Close: 109.0, Volume: 1900.0, High: 110.0, Low: 108.0, Open: 109.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1000.0, 1100.0, 1200.0, 1300.0, 1400.0, 1500.0, 1600.0, 1700.0, 1800.0, 1900.0}

	if len(data.Volume) != len(expectedVolumes) {
		t.Fatalf("Volume length = %d, want %d", len(data.Volume), len(expectedVolumes))
	}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.2f, want %.2f", i, data.Volume[i], expected)
		}
	}
}

// TestCalculateIntradaySeries_ATR14 测试 ATR14 计算
func TestCalculateIntradaySeries_ATR14(t *testing.T) {
	tests := []struct {
		name          string
		klineCount    int
		expectZero    bool
		expectNonZero bool
	}{
		{
			name:          "足够数据 - 20个K线",
			klineCount:    20,
			expectNonZero: true,
		},
		{
			name:          "刚好15个K线（ATR14需要至少15个）",
			klineCount:    15,
			expectNonZero: true,
		},
		{
			name:       "数据不足 - 14个K线",
			klineCount: 14,
			expectZero: true,
		},
		{
			name:       "数据不足 - 10个K线",
			klineCount: 10,
			expectZero: true,
		},
		{
			name:       "数据不足 - 5个K线",
			klineCount: 5,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if tt.expectZero && len(data.ATR14Values) != 0 {
				t.Errorf("ATR14Values length = %d, expected 0 (insufficient data)", len(data.ATR14Values))
			}

			if tt.expectNonZero && len(data.ATR14Values) == 0 {
				t.Errorf("ATR14Values length = 0, expected > 0")
			}

			if tt.expectNonZero && len(data.ATR14Values) > 0 && data.ATR14Values[len(data.ATR14Values)-1] <= 0 {
				t.Errorf("Last ATR14Value = %.3f, expected > 0", data.ATR14Values[len(data.ATR14Values)-1])
			}
		})
	}
}

// TestCalculateATR* 测试已移动到 indicators_test.go

// TestCalculateIntradaySeries_ConsistencyWithOtherIndicators 测试 Volume 和其他指标的一致性
func TestCalculateIntradaySeries_ConsistencyWithOtherIndicators(t *testing.T) {
	klines := generateTestKlines(30)
	data := calculateIntradaySeries(klines)

	// 所有数组应该存在
	if data.MidPrices == nil {
		t.Error("MidPrices should not be nil")
	}
	if data.Volume == nil {
		t.Error("Volume should not be nil")
	}

	// MidPrices 和 Volume 应该有相同的长度（都是最近10个）
	if len(data.MidPrices) != len(data.Volume) {
		t.Errorf("MidPrices length (%d) should equal Volume length (%d)",
			len(data.MidPrices), len(data.Volume))
	}

	// 所有 Volume 值应该大于 0
	for i, vol := range data.Volume {
		if vol <= 0 {
			t.Errorf("Volume[%d] = %.2f, should be > 0", i, vol)
		}
	}
}

// TestCalculateIntradaySeries_EmptyKlines 测试空 K线数据
func TestCalculateIntradaySeries_EmptyKlines(t *testing.T) {
	klines := []Kline{}
	data := calculateIntradaySeries(klines)

	if data == nil {
		t.Fatal("calculateIntradaySeries should not return nil for empty klines")
	}

	// 所有切片应该为空
	if len(data.MidPrices) != 0 {
		t.Errorf("MidPrices length = %d, want 0", len(data.MidPrices))
	}
	if len(data.Volume) != 0 {
		t.Errorf("Volume length = %d, want 0", len(data.Volume))
	}

	// ATR14Values 应该为空（数据不足）
	if len(data.ATR14Values) != 0 {
		t.Errorf("ATR14Values length = %d, want 0", len(data.ATR14Values))
	}
}

// TestCalculateIntradaySeries_VolumePrecision 测试 Volume 精度保持
func TestCalculateIntradaySeries_VolumePrecision(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1234.5678, High: 101.0, Low: 99.0},
		{Close: 101.0, Volume: 9876.5432, High: 102.0, Low: 100.0},
		{Close: 102.0, Volume: 5555.1111, High: 103.0, Low: 101.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1234.5678, 9876.5432, 5555.1111}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.4f, want %.4f (precision not preserved)",
				i, data.Volume[i], expected)
		}
	}
}

// TestIsStaleData_NormalData tests that normal fluctuating data returns false
func TestIsStaleData_NormalData(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.5, Volume: 1200},
		{Close: 99.8, Volume: 900},
		{Close: 100.2, Volume: 1100},
		{Close: 100.1, Volume: 950},
	}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for normal fluctuating data, got true")
	}
}

// TestIsStaleData_PriceFreezeWithZeroVolume tests that frozen price + zero volume returns true
func TestIsStaleData_PriceFreezeWithZeroVolume(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(klines, "DOGEUSDT")

	if !result {
		t.Error("Expected true for frozen price + zero volume, got false")
	}
}

// TestIsStaleData_PriceFreezeWithVolume tests that frozen price but normal volume returns false
func TestIsStaleData_PriceFreezeWithVolume(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.0, Volume: 1200},
		{Close: 100.0, Volume: 900},
		{Close: 100.0, Volume: 1100},
		{Close: 100.0, Volume: 950},
	}

	result := isStaleData(klines, "STABLECOIN")

	if result {
		t.Error("Expected false for frozen price but normal volume (low volatility market), got true")
	}
}

// TestIsStaleData_InsufficientData tests that insufficient data (<2 klines) returns false
func TestIsStaleData_InsufficientData(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for insufficient data (<2 klines), got true")
	}
}

// TestIsStaleData_ExactlyTwoKlines tests edge case with exactly 2 klines (threshold)
func TestIsStaleData_ExactlyTwoKlines(t *testing.T) {
	// Stale case: exactly 2 frozen klines with zero volume
	staleKlines := []Kline{
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(staleKlines, "TESTUSDT")
	if !result {
		t.Error("Expected true for exactly 2 frozen klines with zero volume, got false")
	}

	// Normal case: exactly 2 klines with fluctuation
	normalKlines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.1, Volume: 1100},
	}

	result = isStaleData(normalKlines, "TESTUSDT")
	if result {
		t.Error("Expected false for exactly 2 normal klines, got true")
	}
}

// TestIsStaleData_WithinTolerance tests price changes within tolerance (0.01%)
func TestIsStaleData_WithinTolerance(t *testing.T) {
	// Price changes within 0.01% tolerance should be treated as frozen
	basePrice := 10000.0
	tolerance := 0.0001                        // 0.01%
	smallChange := basePrice * tolerance * 0.5 // Half of tolerance

	klines := []Kline{
		{Close: basePrice, Volume: 1000},
		{Close: basePrice + smallChange, Volume: 1000},
		{Close: basePrice - smallChange, Volume: 1000},
		{Close: basePrice, Volume: 1000},
		{Close: basePrice + smallChange, Volume: 1000},
	}

	result := isStaleData(klines, "BTCUSDT")

	// Should return false because there's normal volume despite tiny price changes
	if result {
		t.Error("Expected false for price within tolerance but with volume, got true")
	}
}

// TestIsStaleData_MixedScenario tests realistic scenario with some history before freeze
func TestIsStaleData_MixedScenario(t *testing.T) {
	// Simulate: normal trading → suddenly freezes
	klines := []Kline{
		{Close: 100.0, Volume: 1000}, // Normal
		{Close: 100.5, Volume: 1200}, // Normal
		{Close: 100.2, Volume: 1100}, // Normal
		{Close: 50.0, Volume: 0},     // Freeze starts
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen (last 5 are all frozen)
	}

	result := isStaleData(klines, "DOGEUSDT")

	// Should detect stale data based on last 5 klines
	if !result {
		t.Error("Expected true for frozen last 5 klines with zero volume, got false")
	}
}

// TestIsStaleData_EmptyKlines tests edge case with empty slice
func TestIsStaleData_EmptyKlines(t *testing.T) {
	klines := []Kline{}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for empty klines, got true")
	}
}

// TestCalculateATRSeries* 测试已移动到 indicators_test.go

// =============================================================================
// ER 和 Bollinger Bands 集成测试
// =============================================================================

// TestCalculateIntradaySeries_ER 测试 ER 数据填充
func TestCalculateIntradaySeries_ER(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		expectNonEmpty bool
	}{
		{
			name:           "足够数据 - 30个K线",
			klineCount:     30,
			expectNonEmpty: true,
		},
		{
			name:           "数据不足 - 10个K线",
			klineCount:     10,
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if tt.expectNonEmpty {
				if len(data.ER10Values) == 0 {
					t.Error("ER10Values should not be empty")
				}
				// ER 值应该在 0-1 范围内
				for i, v := range data.ER10Values {
					if v < 0 || v > 1 {
						t.Errorf("ER10Values[%d] = %.3f, expected in range [0, 1]", i, v)
					}
				}
			} else {
				// 数据不足时返回空切片
				if len(data.ER10Values) != 0 {
					t.Errorf("ER10Values should be empty for insufficient data, got %d", len(data.ER10Values))
				}
			}
		})
	}
}

// TestCalculateIntradaySeries_BollingerBands 测试 Bollinger Bands 数据填充
func TestCalculateIntradaySeries_BollingerBands(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		expectNonEmpty bool
	}{
		{
			name:           "足够数据 - 30个K线",
			klineCount:     30,
			expectNonEmpty: true,
		},
		{
			name:           "数据不足 - 10个K线",
			klineCount:     10,
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if tt.expectNonEmpty {
				if len(data.BollingerBandwidths) == 0 {
					t.Error("BollingerBandwidths should not be empty")
				}
				// Bandwidth 应该 >= 0
				for i, bw := range data.BollingerBandwidths {
					if bw < 0 {
						t.Errorf("BollingerBandwidths[%d] = %.3f, expected >= 0", i, bw)
					}
				}
			} else {
				// 数据不足时返回空切片
				if len(data.BollingerBandwidths) != 0 || len(data.BollingerPercentBs) != 0 {
					t.Errorf("Bollinger slices should be empty for insufficient data")
				}
			}
		})
	}
}

// TestCalculateMidTermSeries15m_ERAndBollinger 测试 15m 数据的 ER 和 BB
func TestCalculateMidTermSeries15m_ERAndBollinger(t *testing.T) {
	klines := generateTestKlines(30)
	data := calculateMidTermSeries15m(klines)

	if data == nil {
		t.Fatal("calculateMidTermSeries15m returned nil")
	}

	// ER10Values 应该是一个切片，每个值在 0-1 范围内
	if len(data.ER10Values) == 0 {
		t.Error("ER10Values is empty")
	}
	for i, er := range data.ER10Values {
		if er < 0 || er > 1 {
			t.Errorf("ER10Values[%d] = %.3f, expected in range [0, 1]", i, er)
		}
	}

	// BollingerBandwidths 应该是一个切片，每个值 > 0
	if len(data.BollingerBandwidths) == 0 {
		t.Error("BollingerBandwidths is empty")
	}
	for i, bw := range data.BollingerBandwidths {
		if bw <= 0 {
			t.Errorf("BollingerBandwidths[%d] = %.3f, expected > 0", i, bw)
		}
	}
}

// TestCalculateMidTermSeries1h_ERAndBollinger 测试 1h 数据的 ER 和 BB
func TestCalculateMidTermSeries1h_ERAndBollinger(t *testing.T) {
	klines := generateTestKlines(30)
	data := calculateMidTermSeries1h(klines)

	if data == nil {
		t.Fatal("calculateMidTermSeries1h returned nil")
	}

	// ER10Values 应该是一个切片，每个值在 0-1 范围内
	if len(data.ER10Values) == 0 {
		t.Error("ER10Values is empty")
	}
	for i, er := range data.ER10Values {
		if er < 0 || er > 1 {
			t.Errorf("ER10Values[%d] = %.3f, expected in range [0, 1]", i, er)
		}
	}

	// BollingerBandwidths 应该是一个切片，每个值 > 0
	if len(data.BollingerBandwidths) == 0 {
		t.Error("BollingerBandwidths is empty")
	}
	for i, bw := range data.BollingerBandwidths {
		if bw <= 0 {
			t.Errorf("BollingerBandwidths[%d] = %.3f, expected > 0", i, bw)
		}
	}
}

// =============================================================================
// Format() 输出测试 - 验证 ER 和 Bollinger Bands 在输出中可见
// =============================================================================

// TestFormat_ContainsERAndBollingerBands 测试 Format() 输出包含 ER 和 Bollinger Bands
func TestFormat_ContainsERAndBollingerBands(t *testing.T) {
	// 创建包含有效 ER 和 BB 数据的 Data 对象
	data := &Data{
		Symbol:        "BTCUSDT",
		CurrentPrice:  50000.0,
		PriceChange1h: 0.5,
		CurrentEMA20:  49500.0,
		CurrentMACD:   100.0,
		CurrentRSI7:   55.0,
		FundingRate:   0.0001,
		IntradaySeries: &IntradayData{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{49000, 49500, 50000},
				ER10Values:          []float64{0.70, 0.72, 0.75}, // 有效 ER 序列
				BollingerPercentBs:  []float64{0.55, 0.58, 0.6},  // 有效 %B 序列
				BollingerBandwidths: []float64{0.04, 0.045, 0.05}, // 有效 Bandwidth 序列
			},
		},
		MidTermSeries15m: &MidTermData15m{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{49000, 49500, 50000},
				ER10Values:          []float64{0.60, 0.63, 0.65},
				BollingerPercentBs:  []float64{0.50, 0.52, 0.55},
				BollingerBandwidths: []float64{0.03, 0.035, 0.04},
			},
		},
		MidTermSeries1h: &MidTermData1h{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{49000, 49500, 50000},
				ER10Values:          []float64{0.50, 0.52, 0.55},
				BollingerPercentBs:  []float64{0.45, 0.48, 0.5},
				BollingerBandwidths: []float64{0.02, 0.025, 0.03},
			},
		},
	}

	output := Format(data, false)

	// 验证 IntradaySeries 的 ER 和 BB 在输出中
	tests := []struct {
		name     string
		contains string
	}{
		{"Intraday ER", "Efficiency Ratio"},
		{"Intraday %B", "Bollinger %B"},
		{"Intraday Bandwidth", "Bandwidth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsSubstr(output, tt.contains) {
				t.Errorf("Format() output missing %q\nOutput:\n%s", tt.contains, output)
			}
		})
	}
}

// TestFormat_SkipsNaNValues 测试 Format() 跳过 NaN 值不输出
func TestFormat_SkipsNaNValues(t *testing.T) {
	// 创建包含 NaN 值的 Data 对象（模拟数据不足情况）
	data := &Data{
		Symbol:        "BTCUSDT",
		CurrentPrice:  50000.0,
		PriceChange1h: 0.5,
		CurrentEMA20:  49500.0,
		CurrentMACD:   100.0,
		CurrentRSI7:   55.0,
		FundingRate:   0.0001,
		IntradaySeries: &IntradayData{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{49000, 49500, 50000},
				ER10Values:          []float64{}, // 空切片，不应输出
				BollingerPercentBs:  []float64{}, // 空切片，不应输出
				BollingerBandwidths: []float64{}, // 空切片，不应输出
			},
		},
	}

	output := Format(data, false)

	// 验证空切片不会导致输出问题
	if containsSubstr(output, "NaN") {
		t.Errorf("Format() output should not contain 'NaN' string\nOutput:\n%s", output)
	}

	// 当 ER 为空时，不应该输出 Efficiency Ratio 行
	if containsSubstr(output, "Efficiency Ratio") {
		t.Errorf("Format() should skip Efficiency Ratio when ER10Values is empty\nOutput:\n%s", output)
	}
}

// containsSubstr 检查字符串是否包含子串（辅助函数）
func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Format() 重构基准测试 - 确保重构后输出不变
// =============================================================================

// TestFormat_SeriesOutputStructure 测试 3 个 Series 块的输出结构
func TestFormat_SeriesOutputStructure(t *testing.T) {
	// 创建完整的测试数据
	data := &Data{
		Symbol:        "BTCUSDT",
		CurrentPrice:  50000.0,
		PriceChange1h: 0.5,
		CurrentEMA20:  49500.0,
		CurrentMACD:   100.0,
		CurrentRSI7:   55.0,
		FundingRate:   0.0001,
		IntradaySeries: &IntradayData{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{49000, 49500, 50000},
				EMA20Values:         []float64{48900, 49400, 49900},
				MACDValues:          []float64{50, 75, 100},
				RSI7Values:          []float64{45, 50, 55},
				RSI14Values:         []float64{48, 52, 54},
				Volume:              []float64{1000, 1100, 1200},
				ATR14Values:         []float64{200, 210, 220},
				ER10Values:          []float64{0.70, 0.72, 0.75},
				BollingerPercentBs:  []float64{0.55, 0.58, 0.6},
				BollingerBandwidths: []float64{0.04, 0.045, 0.05},
			},
		},
		MidTermSeries15m: &MidTermData15m{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{48500, 49000, 49500},
				EMA20Values:         []float64{48400, 48900, 49400},
				MACDValues:          []float64{40, 60, 80},
				RSI7Values:          []float64{42, 48, 52},
				RSI14Values:         []float64{44, 50, 53},
				Volume:              []float64{5000, 5500, 6000},
				ATR14Values:         []float64{300, 310, 320},
				ER10Values:          []float64{0.60, 0.63, 0.65},
				BollingerPercentBs:  []float64{0.50, 0.52, 0.55},
				BollingerBandwidths: []float64{0.03, 0.035, 0.04},
			},
		},
		MidTermSeries1h: &MidTermData1h{
			SeriesFields: SeriesFields{
				MidPrices:           []float64{48000, 48500, 49000},
				EMA20Values:         []float64{47900, 48400, 48900},
				MACDValues:          []float64{30, 50, 70},
				RSI7Values:          []float64{40, 45, 50},
				RSI14Values:         []float64{42, 48, 51},
				Volume:              []float64{20000, 22000, 24000},
				ATR14Values:         []float64{400, 420, 440},
				ER10Values:          []float64{0.50, 0.52, 0.55},
				BollingerPercentBs:  []float64{0.45, 0.48, 0.5},
				BollingerBandwidths: []float64{0.02, 0.025, 0.03},
			},
		},
	}

	output := Format(data, false)

	// 验证每个 Series 块包含所有预期的指标
	expectedPatterns := []struct {
		name    string
		pattern string
	}{
		// IntradaySeries (5m)
		{"5m title", "Intraday series (5‑minute intervals"},
		{"5m mid prices", "Mid prices:"},
		{"5m EMA", "EMA indicators (20‑period):"},
		{"5m MACD", "MACD indicators:"},
		{"5m RSI7", "RSI indicators (7‑Period):"},
		{"5m RSI14", "RSI indicators (14‑Period):"},
		{"5m Volume", "Volume:"},
		{"5m ATR", "ATR (14‑period):"},
		{"5m ER", "Efficiency Ratio (10‑period):"},
		{"5m BB", "Bollinger %B:"},

		// MidTermSeries15m
		{"15m title", "Mid‑term series (15‑minute intervals"},

		// MidTermSeries1h
		{"1h title", "Mid‑term series (1‑hour intervals"},
	}

	for _, tt := range expectedPatterns {
		t.Run(tt.name, func(t *testing.T) {
			if !containsSubstr(output, tt.pattern) {
				t.Errorf("Format() output missing %q", tt.pattern)
			}
		})
	}
}
