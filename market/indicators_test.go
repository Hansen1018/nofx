package market

import (
	"math"
	"testing"
)

// =============================================================================
// Helper 函数
// =============================================================================

// generateTestKlines 生成测试用的 K线数据
func generateTestKlines(count int) []Kline {
	klines := make([]Kline, count)
	for i := 0; i < count; i++ {
		// 生成模拟的价格数据，有一定的波动
		basePrice := 100.0
		variance := float64(i%10) * 0.5
		open := basePrice + variance
		high := open + 1.0
		low := open - 0.5
		close := open + 0.3
		volume := 1000.0 + float64(i*100)

		klines[i] = Kline{
			OpenTime:  int64(i * 300000), // 5分钟间隔
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: int64((i+1)*300000 - 1),
		}
	}
	return klines
}

// =============================================================================
// EMA 测试
// =============================================================================

func TestCalculateEMA(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		period     int
		expectZero bool
	}{
		{
			name:       "正常计算 - 足够数据",
			klines:     generateTestKlines(30),
			period:     20,
			expectZero: false,
		},
		{
			name:       "数据不足",
			klines:     generateTestKlines(10),
			period:     20,
			expectZero: true,
		},
		{
			name:       "空数据",
			klines:     []Kline{},
			period:     20,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ema := calculateEMA(tt.klines, tt.period)
			if tt.expectZero {
				if ema != 0 {
					t.Errorf("calculateEMA() = %.3f, expected 0", ema)
				}
			} else {
				if ema <= 0 {
					t.Errorf("calculateEMA() = %.3f, expected > 0", ema)
				}
			}
		})
	}
}

func TestCalculateEMA_Accuracy(t *testing.T) {
	// 简单的手工验证用例：所有 close 相同，EMA 应该等于 close
	klines := make([]Kline, 10)
	for i := range klines {
		klines[i] = Kline{Close: 100.0}
	}

	ema := calculateEMA(klines, 5)
	if math.Abs(ema-100.0) > 0.0001 {
		t.Errorf("EMA of constant prices should equal the price: got %.4f, want 100.0", ema)
	}
}

// =============================================================================
// MACD 测试
// =============================================================================

func TestCalculateMACD(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		expectZero bool
	}{
		{
			name:       "正常计算 - 足够数据",
			klines:     generateTestKlines(30),
			expectZero: false,
		},
		{
			name:       "数据不足 - 少于26",
			klines:     generateTestKlines(20),
			expectZero: true,
		},
		{
			name:       "空数据",
			klines:     []Kline{},
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macd := calculateMACD(tt.klines)
			if tt.expectZero && macd != 0 {
				t.Errorf("calculateMACD() = %.3f, expected 0", macd)
			}
			// MACD 可以是正、负或零，所以不检查符号
		})
	}
}

func TestCalculateMACD_Accuracy(t *testing.T) {
	// 所有 close 相同，EMA12 == EMA26，MACD 应该为 0
	klines := make([]Kline, 30)
	for i := range klines {
		klines[i] = Kline{Close: 100.0}
	}

	macd := calculateMACD(klines)
	if math.Abs(macd) > 0.0001 {
		t.Errorf("MACD of constant prices should be 0: got %.4f", macd)
	}
}

// =============================================================================
// RSI 测试
// =============================================================================

func TestCalculateRSI(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		period     int
		expectZero bool
	}{
		{
			name:       "正常计算 - 足够数据",
			klines:     generateTestKlines(30),
			period:     14,
			expectZero: false,
		},
		{
			name:       "数据不足",
			klines:     generateTestKlines(10),
			period:     14,
			expectZero: true,
		},
		{
			name:       "空数据",
			klines:     []Kline{},
			period:     14,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsi := calculateRSI(tt.klines, tt.period)
			if tt.expectZero {
				if rsi != 0 {
					t.Errorf("calculateRSI() = %.3f, expected 0", rsi)
				}
			} else {
				// RSI 应该在 0-100 范围内
				if rsi < 0 || rsi > 100 {
					t.Errorf("calculateRSI() = %.3f, expected in range [0, 100]", rsi)
				}
			}
		})
	}
}

func TestCalculateRSI_Extremes(t *testing.T) {
	// 测试全涨情况 - RSI 应该接近 100
	risingKlines := make([]Kline, 20)
	for i := range risingKlines {
		risingKlines[i] = Kline{Close: 100.0 + float64(i)}
	}

	rsi := calculateRSI(risingKlines, 14)
	if rsi < 90 {
		t.Errorf("RSI of rising prices should be close to 100: got %.2f", rsi)
	}

	// 测试全跌情况 - RSI 应该接近 0
	fallingKlines := make([]Kline, 20)
	for i := range fallingKlines {
		fallingKlines[i] = Kline{Close: 100.0 - float64(i)}
	}

	rsi = calculateRSI(fallingKlines, 14)
	if rsi > 10 {
		t.Errorf("RSI of falling prices should be close to 0: got %.2f", rsi)
	}
}

// =============================================================================
// ATR 测试
// =============================================================================

// TestCalculateATR 测试 ATR 计算函数
func TestCalculateATR(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		period     int
		expectZero bool
	}{
		{
			name: "正常计算 - 足够数据",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
				{High: 104.0, Low: 102.0, Close: 103.0},
				{High: 105.0, Low: 103.0, Close: 104.0},
				{High: 106.0, Low: 104.0, Close: 105.0},
				{High: 107.0, Low: 105.0, Close: 106.0},
				{High: 108.0, Low: 106.0, Close: 107.0},
				{High: 109.0, Low: 107.0, Close: 108.0},
				{High: 110.0, Low: 108.0, Close: 109.0},
				{High: 111.0, Low: 109.0, Close: 110.0},
				{High: 112.0, Low: 110.0, Close: 111.0},
				{High: 113.0, Low: 111.0, Close: 112.0},
				{High: 114.0, Low: 112.0, Close: 113.0},
				{High: 115.0, Low: 113.0, Close: 114.0},
				{High: 116.0, Low: 114.0, Close: 115.0},
			},
			period:     14,
			expectZero: false,
		},
		{
			name: "数据不足 - 等于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
			},
			period:     2,
			expectZero: true,
		},
		{
			name: "数据不足 - 少于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
			},
			period:     14,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atr := calculateATR(tt.klines, tt.period)

			if tt.expectZero {
				if atr != 0 {
					t.Errorf("calculateATR() = %.3f, expected 0 (insufficient data)", atr)
				}
			} else {
				if atr <= 0 {
					t.Errorf("calculateATR() = %.3f, expected > 0", atr)
				}
			}
		})
	}
}

// TestCalculateATR_TrueRange 测试 ATR 的 True Range 计算正确性
func TestCalculateATR_TrueRange(t *testing.T) {
	// 创建一个简单的测试用例，手动计算期望的 ATR
	klines := []Kline{
		{High: 50.0, Low: 48.0, Close: 49.0}, // TR = 2.0
		{High: 51.0, Low: 49.0, Close: 50.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 52.0, Low: 50.0, Close: 51.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 53.0, Low: 51.0, Close: 52.0}, // TR = 2.0
		{High: 54.0, Low: 52.0, Close: 53.0}, // TR = 2.0
	}

	atr := calculateATR(klines, 3)

	// 期望的计算：
	// TR[1] = max(51-49, |51-49|, |49-49|) = 2.0
	// TR[2] = max(52-50, |52-50|, |50-50|) = 2.0
	// TR[3] = max(53-51, |53-51|, |51-51|) = 2.0
	// 初始 ATR = (2.0 + 2.0 + 2.0) / 3 = 2.0
	// TR[4] = max(54-52, |54-52|, |52-52|) = 2.0
	// 平滑 ATR = (2.0*2 + 2.0) / 3 = 2.0

	expectedATR := 2.0
	tolerance := 0.01 // 允许小的浮点误差

	if math.Abs(atr-expectedATR) > tolerance {
		t.Errorf("calculateATR() = %.3f, want approximately %.3f", atr, expectedATR)
	}
}

// =============================================================================
// ATR Series 测试
// =============================================================================

// TestCalculateATRSeries 测试 ATR 序列计算
func TestCalculateATRSeries(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		period         int
		expectedLen    int
		expectNonEmpty bool
	}{
		{
			name:           "足够数据 - 100根K线, period=14",
			klineCount:     100,
			period:         14,
			expectedLen:    10, // 最多返回10个点
			expectNonEmpty: true,
		},
		{
			name:           "刚好足够 - 24根K线 (14+10), period=14",
			klineCount:     24,
			period:         14,
			expectedLen:    10,
			expectNonEmpty: true,
		},
		{
			name:           "部分数据 - 20根K线, period=14",
			klineCount:     20,
			period:         14,
			expectedLen:    6, // 20 - 14 = 6个点
			expectNonEmpty: true,
		},
		{
			name:           "最少数据 - 15根K线, period=14",
			klineCount:     15,
			period:         14,
			expectedLen:    1, // 只有1个ATR值
			expectNonEmpty: true,
		},
		{
			name:           "数据不足 - 14根K线, period=14",
			klineCount:     14,
			period:         14,
			expectedLen:    0,
			expectNonEmpty: false,
		},
		{
			name:           "数据不足 - 10根K线, period=14",
			klineCount:     10,
			period:         14,
			expectedLen:    0,
			expectNonEmpty: false,
		},
		{
			name:           "空数据",
			klineCount:     0,
			period:         14,
			expectedLen:    0,
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			atrValues := calculateATRSeries(klines, tt.period)

			if len(atrValues) != tt.expectedLen {
				t.Errorf("calculateATRSeries() length = %d, want %d", len(atrValues), tt.expectedLen)
			}

			if tt.expectNonEmpty {
				for i, v := range atrValues {
					if v <= 0 {
						t.Errorf("ATR[%d] = %.3f, expected > 0", i, v)
					}
				}
			}
		})
	}
}

// TestCalculateATRSeries_ValuesConsistency 测试 ATR 序列值与单值计算的一致性
func TestCalculateATRSeries_ValuesConsistency(t *testing.T) {
	klines := generateTestKlines(30)

	// 序列最后一个值应该等于单值计算的结果
	atrSeries := calculateATRSeries(klines, 14)
	atrSingle := calculateATR(klines, 14)

	if len(atrSeries) == 0 {
		t.Fatal("ATR series should not be empty")
	}

	lastValue := atrSeries[len(atrSeries)-1]
	tolerance := 0.0001

	if math.Abs(lastValue-atrSingle) > tolerance {
		t.Errorf("Last ATR series value (%.6f) should equal single ATR (%.6f)",
			lastValue, atrSingle)
	}
}

// TestCalculateATRSeries_TrendDetection 测试 ATR 序列能够检测趋势
func TestCalculateATRSeries_TrendDetection(t *testing.T) {
	// 创建波动率递增的K线数据
	expandingKlines := make([]Kline, 30)
	for i := 0; i < 30; i++ {
		basePrice := 100.0
		// 波动率随时间增加
		volatility := 1.0 + float64(i)*0.1
		expandingKlines[i] = Kline{
			High:  basePrice + volatility,
			Low:   basePrice - volatility,
			Close: basePrice,
		}
	}

	atrSeries := calculateATRSeries(expandingKlines, 14)

	if len(atrSeries) < 2 {
		t.Fatal("ATR series should have at least 2 values for trend detection")
	}

	// 验证 ATR 序列整体呈递增趋势
	firstHalf := atrSeries[:len(atrSeries)/2]
	secondHalf := atrSeries[len(atrSeries)/2:]

	avgFirst := 0.0
	for _, v := range firstHalf {
		avgFirst += v
	}
	avgFirst /= float64(len(firstHalf))

	avgSecond := 0.0
	for _, v := range secondHalf {
		avgSecond += v
	}
	avgSecond /= float64(len(secondHalf))

	if avgSecond <= avgFirst {
		t.Errorf("Expanding volatility should produce increasing ATR: first half avg=%.3f, second half avg=%.3f",
			avgFirst, avgSecond)
	}
}

// =============================================================================
// Efficiency Ratio (ER) 测试
// =============================================================================

func TestCalculateEfficiencyRatio(t *testing.T) {
	tests := []struct {
		name          string
		klines        []Kline
		period        int
		expectInvalid bool // 数据不足时期望返回 -1
	}{
		{
			name:          "正常计算 - 足够数据",
			klines:        generateTestKlines(20),
			period:        10,
			expectInvalid: false,
		},
		{
			name:          "数据不足",
			klines:        generateTestKlines(5),
			period:        10,
			expectInvalid: true,
		},
		{
			name:          "空数据",
			klines:        []Kline{},
			period:        10,
			expectInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := calculateEfficiencyRatio(tt.klines, tt.period)
			if tt.expectInvalid {
				if !math.IsNaN(er) {
					t.Errorf("calculateEfficiencyRatio() = %.3f, expected NaN (invalid)", er)
				}
			} else {
				// ER 应该在 0-1 范围内
				if er < 0 || er > 1 {
					t.Errorf("calculateEfficiencyRatio() = %.3f, expected in range [0, 1]", er)
				}
			}
		})
	}
}

func TestCalculateEfficiencyRatio_PerfectTrend(t *testing.T) {
	// 完美趋势：每天涨1元，ER应该接近1.0
	klines := make([]Kline, 15)
	for i := range klines {
		klines[i] = Kline{Close: 100.0 + float64(i)}
	}

	er := calculateEfficiencyRatio(klines, 10)

	// 完美趋势 ER = 1.0
	if math.Abs(er-1.0) > 0.0001 {
		t.Errorf("ER of perfect trend should be 1.0: got %.4f", er)
	}
}

func TestCalculateEfficiencyRatio_Choppy(t *testing.T) {
	// 震荡行情：涨跌交替，净移动为0，ER应该接近0
	klines := make([]Kline, 15)
	for i := range klines {
		if i%2 == 0 {
			klines[i] = Kline{Close: 100.0}
		} else {
			klines[i] = Kline{Close: 101.0}
		}
	}

	er := calculateEfficiencyRatio(klines, 10)

	// 震荡行情 ER 应该很低 (接近0)
	if er > 0.2 {
		t.Errorf("ER of choppy market should be close to 0: got %.4f", er)
	}
}

// =============================================================================
// Bollinger Bands 测试
// =============================================================================

func TestCalculateBollingerBands(t *testing.T) {
	tests := []struct {
		name          string
		klines        []Kline
		period        int
		expectInvalid bool // 数据不足时期望返回 -1, -1
	}{
		{
			name:          "正常计算 - 足够数据",
			klines:        generateTestKlines(30),
			period:        20,
			expectInvalid: false,
		},
		{
			name:          "数据不足",
			klines:        generateTestKlines(10),
			period:        20,
			expectInvalid: true,
		},
		{
			name:          "空数据",
			klines:        []Kline{},
			period:        20,
			expectInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentB, bandwidth := calculateBollingerBands(tt.klines, tt.period, 2.0)
			if tt.expectInvalid {
				if !math.IsNaN(percentB) || !math.IsNaN(bandwidth) {
					t.Errorf("calculateBollingerBands() = (%.3f, %.3f), expected (NaN, NaN) (invalid)", percentB, bandwidth)
				}
			} else {
				// Bandwidth 应该 > 0（有波动的数据）
				if bandwidth <= 0 {
					t.Errorf("Bandwidth = %.3f, expected > 0", bandwidth)
				}
			}
		})
	}
}

func TestCalculateBollingerBands_ConstantPrice(t *testing.T) {
	// 价格恒定：标准差为0，带宽为0，%B = 0.5（中间）
	klines := make([]Kline, 25)
	for i := range klines {
		klines[i] = Kline{Close: 100.0}
	}

	percentB, bandwidth := calculateBollingerBands(klines, 20, 2.0)

	// 恒定价格时，标准差为0，带宽为0
	if bandwidth != 0 {
		t.Errorf("Bandwidth of constant prices should be 0: got %.4f", bandwidth)
	}

	// %B 应该是 0.5（价格在中轨，即SMA上）
	// 但当带宽为0时，%B 的定义需要特殊处理
	if percentB != 0.5 {
		t.Errorf("PercentB of constant prices should be 0.5: got %.4f", percentB)
	}
}

func TestCalculateBollingerBands_PercentBRange(t *testing.T) {
	// 正常波动数据，%B 通常在 0-1 附近，但可以超出
	klines := generateTestKlines(30)

	percentB, bandwidth := calculateBollingerBands(klines, 20, 2.0)

	// 带宽应该 > 0
	if bandwidth <= 0 {
		t.Errorf("Bandwidth should be > 0 for fluctuating prices: got %.4f", bandwidth)
	}

	// %B 一般在 -0.5 到 1.5 范围内（正常情况下在 0-1）
	if percentB < -1 || percentB > 2 {
		t.Errorf("PercentB = %.3f seems out of reasonable range", percentB)
	}
}

// =============================================================================
// ER Series 测试
// =============================================================================

// TestCalculateERSeries 测试 ER 序列计算
func TestCalculateERSeries(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		period         int
		expectedLen    int
		expectNonEmpty bool
	}{
		{
			name:           "足够数据 - 100根K线, period=10",
			klineCount:     100,
			period:         10,
			expectedLen:    10, // 最多返回10个点
			expectNonEmpty: true,
		},
		{
			name:           "刚好足够 - 20根K线 (10+10), period=10",
			klineCount:     20,
			period:         10,
			expectedLen:    10,
			expectNonEmpty: true,
		},
		{
			name:           "部分数据 - 15根K线, period=10",
			klineCount:     15,
			period:         10,
			expectedLen:    5, // 15 - 10 = 5个点
			expectNonEmpty: true,
		},
		{
			name:           "最少数据 - 11根K线, period=10",
			klineCount:     11,
			period:         10,
			expectedLen:    1, // 只有1个ER值
			expectNonEmpty: true,
		},
		{
			name:           "数据不足 - 10根K线, period=10",
			klineCount:     10,
			period:         10,
			expectedLen:    0,
			expectNonEmpty: false,
		},
		{
			name:           "空数据",
			klineCount:     0,
			period:         10,
			expectedLen:    0,
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			erValues := calculateERSeries(klines, tt.period)

			if len(erValues) != tt.expectedLen {
				t.Errorf("calculateERSeries() length = %d, want %d", len(erValues), tt.expectedLen)
			}

			if tt.expectNonEmpty {
				for i, v := range erValues {
					// ER 应该在 0-1 范围内
					if v < 0 || v > 1 {
						t.Errorf("ER[%d] = %.3f, expected in range [0, 1]", i, v)
					}
				}
			}
		})
	}
}

// TestCalculateERSeries_ValuesConsistency 测试 ER 序列值与单值计算的一致性
func TestCalculateERSeries_ValuesConsistency(t *testing.T) {
	klines := generateTestKlines(30)

	// 序列最后一个值应该等于单值计算的结果
	erSeries := calculateERSeries(klines, 10)
	erSingle := calculateEfficiencyRatio(klines, 10)

	if len(erSeries) == 0 {
		t.Fatal("ER series should not be empty")
	}

	lastValue := erSeries[len(erSeries)-1]
	tolerance := 0.0001

	if math.Abs(lastValue-erSingle) > tolerance {
		t.Errorf("Last ER series value (%.6f) should equal single ER (%.6f)",
			lastValue, erSingle)
	}
}

// =============================================================================
// Bollinger Bands Series 测试
// =============================================================================

// TestCalculateBollingerSeries 测试 Bollinger Bands 序列计算
func TestCalculateBollingerSeries(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		period         int
		expectedLen    int
		expectNonEmpty bool
	}{
		{
			name:           "足够数据 - 100根K线, period=20",
			klineCount:     100,
			period:         20,
			expectedLen:    10, // 最多返回10个点
			expectNonEmpty: true,
		},
		{
			name:           "刚好足够 - 30根K线 (20+10), period=20",
			klineCount:     30,
			period:         20,
			expectedLen:    10,
			expectNonEmpty: true,
		},
		{
			name:           "部分数据 - 25根K线, period=20",
			klineCount:     25,
			period:         20,
			expectedLen:    6, // 25 - 20 + 1 = 6个点
			expectNonEmpty: true,
		},
		{
			name:           "最少数据 - 20根K线, period=20",
			klineCount:     20,
			period:         20,
			expectedLen:    1, // 只有1个值
			expectNonEmpty: true,
		},
		{
			name:           "数据不足 - 19根K线, period=20",
			klineCount:     19,
			period:         20,
			expectedLen:    0,
			expectNonEmpty: false,
		},
		{
			name:           "空数据",
			klineCount:     0,
			period:         20,
			expectedLen:    0,
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			percentBValues, bandwidthValues := calculateBollingerSeries(klines, tt.period, 2.0)

			if len(percentBValues) != tt.expectedLen {
				t.Errorf("calculateBollingerSeries() percentB length = %d, want %d", len(percentBValues), tt.expectedLen)
			}
			if len(bandwidthValues) != tt.expectedLen {
				t.Errorf("calculateBollingerSeries() bandwidth length = %d, want %d", len(bandwidthValues), tt.expectedLen)
			}

			if tt.expectNonEmpty {
				for i, bw := range bandwidthValues {
					// Bandwidth 应该 >= 0
					if bw < 0 {
						t.Errorf("Bandwidth[%d] = %.3f, expected >= 0", i, bw)
					}
				}
			}
		})
	}
}

// TestCalculateBollingerSeries_ValuesConsistency 测试 Bollinger 序列值与单值计算的一致性
func TestCalculateBollingerSeries_ValuesConsistency(t *testing.T) {
	klines := generateTestKlines(30)

	// 序列最后一个值应该等于单值计算的结果
	percentBSeries, bandwidthSeries := calculateBollingerSeries(klines, 20, 2.0)
	percentBSingle, bandwidthSingle := calculateBollingerBands(klines, 20, 2.0)

	if len(percentBSeries) == 0 {
		t.Fatal("Bollinger series should not be empty")
	}

	tolerance := 0.0001

	lastPercentB := percentBSeries[len(percentBSeries)-1]
	if math.Abs(lastPercentB-percentBSingle) > tolerance {
		t.Errorf("Last percentB series value (%.6f) should equal single percentB (%.6f)",
			lastPercentB, percentBSingle)
	}

	lastBandwidth := bandwidthSeries[len(bandwidthSeries)-1]
	if math.Abs(lastBandwidth-bandwidthSingle) > tolerance {
		t.Errorf("Last bandwidth series value (%.6f) should equal single bandwidth (%.6f)",
			lastBandwidth, bandwidthSingle)
	}
}
