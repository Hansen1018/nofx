package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FundingRateCache 资金费率缓存结构
// Binance Funding Rate 每 8 小时才更新一次，使用 1 小时缓存可显著减少 API 调用
type FundingRateCache struct {
	Rate      float64
	UpdatedAt time.Time
}

var (
	fundingRateMap sync.Map // map[string]*FundingRateCache
	frCacheTTL     = 1 * time.Hour
)

const (
	DailyInterval   = "1d"
	DailyDataPoints = 7
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines5m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取5分钟K线数据 (缓存中约100根，用于计算指标)
	klines5m, err = WSMonitorCli.GetCurrentKlines(symbol, "5m")
	if err != nil {
		return nil, fmt.Errorf("获取5分钟K线失败: %v", err)
	}

	// Data staleness detection: Prevent DOGEUSDT-style price freeze issues
	if isStaleData(klines5m, symbol) {
		log.Printf("⚠️  WARNING: %s detected stale data (consecutive price freeze), skipping symbol", symbol)
		return nil, fmt.Errorf("%s data is stale, possible cache failure", symbol)
	}

	// 获取15分钟K线数据 (缓存中约100根)
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据 (缓存中约100根)
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据 (缓存中约100根)
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h")
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 检查数据是否为空
	if len(klines5m) == 0 {
		return nil, fmt.Errorf("5分钟K线数据为空")
	}
	if len(klines4h) == 0 {
		return nil, fmt.Errorf("4小时K线数据为空")
	}
	if len(klines15m) == 0 {
		return nil, fmt.Errorf("15分钟K线数据为空")
	}
	if len(klines1h) == 0 {
		return nil, fmt.Errorf("1小时K线数据为空")
	}

	// 计算当前指标 (基于5分钟最新数据)
	currentPrice := klines5m[len(klines5m)-1].Close
	currentEMA20 := calculateEMA(klines5m, 20)
	currentMACD := calculateMACD(klines5m)
	currentRSI7 := calculateRSI(klines5m, 7)

	// 计算价格变化百分比
	// 1小时价格变化 = 12个5分钟K线前的价格
	priceChange1h := 0.0
	if len(klines5m) >= 13 { // 至少需要13根K线 (当前 + 12根前)
		price1hAgo := klines5m[len(klines5m)-13].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取日线K线数据用于计算24小时价格变化
	klines1d, err := WSMonitorCli.GetCurrentKlines(symbol, "1d")
	if err != nil {
		log.Printf("获取日线K线失败: %v", err)
	}

	// 24小时价格变化 = 使用日线K线计算
	priceChange24h := 0.0
	if len(klines1d) >= 2 {
		price24hAgo := klines1d[len(klines1d)-2].Close
		if price24hAgo > 0 {
			priceChange24h = ((currentPrice - price24hAgo) / price24hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据
	intradayData := calculateIntradaySeries(klines5m)

	// 计算中期系列数据 - 15分钟
	midTermData15m := calculateMidTermSeries15m(klines15m)

	// 计算中期系列数据 - 1小时
	midTermData1h := calculateMidTermSeries1h(klines1h)

	// 计算长期数据
	longerTermData := calculateLongerTermData(klines4h)

	// 获取日线数据
	dailyData, err := getDailyData(symbol)
	if err != nil {
		// 日线数据失败不应该阻塞主要流程，记录错误即可
		log.Printf("获取日线数据失败: %v", err)
	}

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		PriceChange24h:    priceChange24h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		MidTermSeries15m:  midTermData15m,
		MidTermSeries1h:   midTermData1h,
		LongerTermContext: longerTermData,
		DailyContext:      dailyData,
	}, nil
}

// =============================================================================
// 时间序列指标计算（5m/15m/1h 通用）
// =============================================================================

// seriesResult 内部计算结果，用于填充各周期数据结构
type seriesResult struct {
	midPrices           []float64
	ema20Values         []float64
	macdValues          []float64
	rsi7Values          []float64
	rsi14Values         []float64
	volume              []float64
	atr14Values         []float64
	er10Values          []float64
	bollingerPercentBs  []float64
	bollingerBandwidths []float64
}

// calculateSeriesData 计算时间序列指标（5m/15m/1h 通用）
func calculateSeriesData(klines []Kline) *seriesResult {
	r := &seriesResult{
		midPrices:   make([]float64, 0, 10),
		ema20Values: make([]float64, 0, 10),
		macdValues:  make([]float64, 0, 10),
		rsi7Values:  make([]float64, 0, 10),
		rsi14Values: make([]float64, 0, 10),
		volume:      make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		r.midPrices = append(r.midPrices, klines[i].Close)
		r.volume = append(r.volume, klines[i].Volume)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			r.ema20Values = append(r.ema20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			r.macdValues = append(r.macdValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			r.rsi7Values = append(r.rsi7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			r.rsi14Values = append(r.rsi14Values, rsi14)
		}
	}

	// 计算 ATR14 序列
	r.atr14Values = calculateATRSeries(klines, 14)

	// 计算 Efficiency Ratio (10期) 序列
	r.er10Values = calculateERSeries(klines, 10)

	// 计算 Bollinger Bands (20期, 2倍标准差) 序列
	r.bollingerPercentBs, r.bollingerBandwidths = calculateBollingerSeries(klines, 20, 2.0)

	return r
}

// calculateIntradaySeries 计算日内系列数据 (5m)
func calculateIntradaySeries(klines []Kline) *IntradayData {
	r := calculateSeriesData(klines)
	return &IntradayData{
		SeriesFields: SeriesFields{
			MidPrices:           r.midPrices,
			EMA20Values:         r.ema20Values,
			MACDValues:          r.macdValues,
			RSI7Values:          r.rsi7Values,
			RSI14Values:         r.rsi14Values,
			Volume:              r.volume,
			ATR14Values:         r.atr14Values,
			ER10Values:          r.er10Values,
			BollingerPercentBs:  r.bollingerPercentBs,
			BollingerBandwidths: r.bollingerBandwidths,
		},
	}
}

// calculateMidTermSeries15m 计算15分钟中期系列数据
func calculateMidTermSeries15m(klines []Kline) *MidTermData15m {
	r := calculateSeriesData(klines)
	return &MidTermData15m{
		SeriesFields: SeriesFields{
			MidPrices:           r.midPrices,
			EMA20Values:         r.ema20Values,
			MACDValues:          r.macdValues,
			RSI7Values:          r.rsi7Values,
			RSI14Values:         r.rsi14Values,
			Volume:              r.volume,
			ATR14Values:         r.atr14Values,
			ER10Values:          r.er10Values,
			BollingerPercentBs:  r.bollingerPercentBs,
			BollingerBandwidths: r.bollingerBandwidths,
		},
	}
}

// calculateMidTermSeries1h 计算1小时中期系列数据
func calculateMidTermSeries1h(klines []Kline) *MidTermData1h {
	r := calculateSeriesData(klines)
	return &MidTermData1h{
		SeriesFields: SeriesFields{
			MidPrices:           r.midPrices,
			EMA20Values:         r.ema20Values,
			MACDValues:          r.macdValues,
			RSI7Values:          r.rsi7Values,
			RSI14Values:         r.rsi14Values,
			Volume:              r.volume,
			ATR14Values:         r.atr14Values,
			ER10Values:          r.er10Values,
			BollingerPercentBs:  r.bollingerPercentBs,
			BollingerBandwidths: r.bollingerBandwidths,
		},
	}
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14Values = calculateATRSeries(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算 Efficiency Ratio (10期) 序列
	data.ER10Values = calculateERSeries(klines, 10)

	// 计算 Bollinger Bands (20期, 2倍标准差) 序列
	data.BollingerPercentBs, data.BollingerBandwidths = calculateBollingerSeries(klines, 20, 2.0)

	return data
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率（优化：使用 1 小时缓存）
func getFundingRate(symbol string) (float64, error) {
	// 检查缓存（有效期 1 小时）
	// Funding Rate 每 8 小时才更新，1 小时缓存非常合理
	if cached, ok := fundingRateMap.Load(symbol); ok {
		cache := cached.(*FundingRateCache)
		if time.Since(cache.UpdatedAt) < frCacheTTL {
			// 缓存命中，直接返回
			return cache.Rate, nil
		}
	}

	// 缓存过期或不存在，调用 API
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)

	// 更新缓存
	fundingRateMap.Store(symbol, &FundingRateCache{
		Rate:      rate,
		UpdatedAt: time.Now(),
	})

	return rate, nil
}

func getDailyData(symbol string) (*DailyData, error) {
	// 获取全部 klines 用于计算指标
	fullKlines, err := WSMonitorCli.GetCurrentKlines(symbol, DailyInterval)
	if err != nil {
		return nil, err
	}

	// 截取最后 7 根用于显示
	klines := fullKlines
	if len(klines) > DailyDataPoints {
		klines = klines[len(klines)-DailyDataPoints:]
	}

	data := &DailyData{
		Dates:       make([]string, len(klines)),
		OpenPrices:  make([]float64, len(klines)),
		HighPrices:  make([]float64, len(klines)),
		LowPrices:   make([]float64, len(klines)),
		ClosePrices: make([]float64, len(klines)),
		Volume:      make([]float64, len(klines)),
	}

	// 填充数据
	maxHigh := -math.MaxFloat64
	minLow := math.MaxFloat64

	for i, k := range klines {
		data.Dates[i] = time.UnixMilli(k.OpenTime).Format("2006-01-02")
		data.OpenPrices[i] = k.Open
		data.HighPrices[i] = k.High
		data.LowPrices[i] = k.Low
		data.ClosePrices[i] = k.Close
		data.Volume[i] = k.Volume

		if k.High > maxHigh {
			maxHigh = k.High
		}
		if k.Low < minLow {
			minLow = k.Low
		}
	}

	// 计算指标 (基于全部数据)
	data.EMA20Values = make([]float64, len(klines))
	data.EMA50Values = make([]float64, len(klines))
	data.MACDValues = make([]float64, len(klines))
	data.RSI14Values = make([]float64, len(klines))

	// 计算指标需要使用 fullKlines
	// 我们需要对应 klines 中的每个点，计算其在 fullKlines 中的指标值
	startIdx := len(fullKlines) - len(klines)

	for i := 0; i < len(klines); i++ {
		fullIdx := startIdx + i
		if fullIdx >= 19 {
			data.EMA20Values[i] = calculateEMA(fullKlines[:fullIdx+1], 20)
		}
		if fullIdx >= 49 {
			data.EMA50Values[i] = calculateEMA(fullKlines[:fullIdx+1], 50)
		}
		if fullIdx >= 25 {
			data.MACDValues[i] = calculateMACD(fullKlines[:fullIdx+1])
		}
		if fullIdx >= 14 {
			data.RSI14Values[i] = calculateRSI(fullKlines[:fullIdx+1], 14)
		}
	}

	data.ATR14Values = calculateATRSeries(fullKlines, 14)

	// 计算 Efficiency Ratio (10期) 序列
	data.ER10Values = calculateERSeries(fullKlines, 10)

	// 计算 Bollinger Bands (20期, 2倍标准差) 序列
	data.BollingerPercentBs, data.BollingerBandwidths = calculateBollingerSeries(fullKlines, 20, 2.0)

	// 计算关键价位 (基于最近7根)
	data.Recent7High = maxHigh
	data.Recent7Low = minLow

	// 判断趋势
	if len(data.ClosePrices) > 0 {
		lastIdx := len(data.ClosePrices) - 1
		currentPrice := data.ClosePrices[lastIdx]
		ema20 := data.EMA20Values[lastIdx]
		ema50 := data.EMA50Values[lastIdx]

		// 确保 EMA 值有效 (非0)
		if ema20 > 0 && ema50 > 0 {
			if currentPrice > ema20 && ema20 > ema50 {
				data.TrendBias = "bullish"
			} else if currentPrice < ema20 && ema20 < ema50 {
				data.TrendBias = "bearish"
			} else {
				data.TrendBias = "neutral"
			}
		} else {
			data.TrendBias = "neutral" // 数据不足以计算趋势
		}
	}

	return data, nil
}

// Format 格式化市场数据为字符串
// skipSymbolMention: 如果为 true，在描述 OI/Funding 时不提及币种名称（避免重复）
func Format(data *Data, skipSymbolMention bool) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	sb.WriteString(fmt.Sprintf("current_price = %s, price_change_1h = %.2f%%, price_change_4h = %.2f%%, price_change_24h = %.2f%%\n\n",
		priceStr, data.PriceChange1h, data.PriceChange4h, data.PriceChange24h))
	sb.WriteString(fmt.Sprintf("current_ema20 = %.3f, current_macd = %.3f, current_rsi (7 period) = %.3f\n\n",
		data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	if skipSymbolMention {
		sb.WriteString("Here is the latest open interest and funding rate for perps:\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
			data.Symbol))
	}

	if data.OpenInterest != nil {
		// 使用动态精度格式化 OI 数据
		oiLatestStr := formatPriceWithDynamicPrecision(data.OpenInterest.Latest)
		oiAverageStr := formatPriceWithDynamicPrecision(data.OpenInterest.Average)
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %s Average: %s\n\n",
			oiLatestStr, oiAverageStr))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		formatSeriesData(&sb, "Intraday series (5‑minute intervals, oldest → latest):", &data.IntradaySeries.SeriesFields)
	}

	if data.MidTermSeries15m != nil {
		formatSeriesData(&sb, "Mid‑term series (15‑minute intervals, oldest → latest):", &data.MidTermSeries15m.SeriesFields)
	}

	if data.MidTermSeries1h != nil {
		formatSeriesData(&sb, "Mid‑term series (1‑hour intervals, oldest → latest):", &data.MidTermSeries1h.SeriesFields)
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

		sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f\n\n", data.LongerTermContext.ATR3))

		if len(data.LongerTermContext.ATR14Values) > 0 {
			sb.WriteString(fmt.Sprintf("ATR (14‑period): %s\n\n", formatFloatSlice(data.LongerTermContext.ATR14Values)))
		}

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}

		if len(data.LongerTermContext.ER10Values) > 0 {
			sb.WriteString(fmt.Sprintf("Efficiency Ratio (10‑period): %s\n\n", formatFloatSlice(data.LongerTermContext.ER10Values)))
		}

		if len(data.LongerTermContext.BollingerPercentBs) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger %%B: %s\n\n", formatFloatSlice(data.LongerTermContext.BollingerPercentBs)))
		}

		if len(data.LongerTermContext.BollingerBandwidths) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger Bandwidth: %s\n\n", formatFloatSlice(data.LongerTermContext.BollingerBandwidths)))
		}
	}

	if data.DailyContext != nil {
		sb.WriteString("\nDaily context (last 7 days):\n\n")

		// 趋势摘要
		sb.WriteString(fmt.Sprintf("Trend bias: %s\n", data.DailyContext.TrendBias))
		sb.WriteString(fmt.Sprintf("7-day range: %.2f - %.2f\n", data.DailyContext.Recent7Low, data.DailyContext.Recent7High))

		if len(data.DailyContext.ClosePrices) > 0 {
			lastIdx := len(data.DailyContext.ClosePrices) - 1
			sb.WriteString(fmt.Sprintf("Current vs EMA20: %.2f vs %.2f\n",
				data.DailyContext.ClosePrices[lastIdx],
				data.DailyContext.EMA20Values[lastIdx]))
			sb.WriteString(fmt.Sprintf("EMA20 vs EMA50: %.2f vs %.2f\n",
				data.DailyContext.EMA20Values[lastIdx],
				data.DailyContext.EMA50Values[lastIdx]))
		}

		if len(data.DailyContext.ATR14Values) > 0 {
			sb.WriteString(fmt.Sprintf("Daily ATR (14): %s\n\n", formatFloatSlice(data.DailyContext.ATR14Values)))
		}

		// 最近5天的OHLC
		sb.WriteString("Recent 5 days OHLC:\n")
		start := len(data.DailyContext.Dates) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(data.DailyContext.Dates); i++ {
			sb.WriteString(fmt.Sprintf("  %s: O=%.2f H=%.2f L=%.2f C=%.2f\n",
				data.DailyContext.Dates[i],
				data.DailyContext.OpenPrices[i],
				data.DailyContext.HighPrices[i],
				data.DailyContext.LowPrices[i],
				data.DailyContext.ClosePrices[i]))
		}

		// MACD序列（最近10天）
		if len(data.DailyContext.MACDValues) > 0 {
			startMACD := len(data.DailyContext.MACDValues) - 10
			if startMACD < 0 {
				startMACD = 0
			}
			sb.WriteString(fmt.Sprintf("\nDaily MACD (last 10): %v\n",
				formatFloatSlice(data.DailyContext.MACDValues[startMACD:])))
		}

		// RSI序列（最近10天）
		if len(data.DailyContext.RSI14Values) > 0 {
			startRSI := len(data.DailyContext.RSI14Values) - 10
			if startRSI < 0 {
				startRSI = 0
			}
			sb.WriteString(fmt.Sprintf("\nDaily RSI14 (last 10): %v\n",
				formatFloatSlice(data.DailyContext.RSI14Values[startRSI:])))
		}

		if len(data.DailyContext.ER10Values) > 0 {
			sb.WriteString(fmt.Sprintf("\nDaily ER (10‑period): %s\n", formatFloatSlice(data.DailyContext.ER10Values)))
		}

		if len(data.DailyContext.BollingerPercentBs) > 0 {
			sb.WriteString(fmt.Sprintf("Daily Bollinger %%B: %s\n", formatFloatSlice(data.DailyContext.BollingerPercentBs)))
		}

		if len(data.DailyContext.BollingerBandwidths) > 0 {
			sb.WriteString(fmt.Sprintf("Daily Bollinger Bandwidth: %s\n", formatFloatSlice(data.DailyContext.BollingerBandwidths)))
		}
	}

	return sb.String()
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatSeriesData 通用时序数据格式化函数
func formatSeriesData(sb *strings.Builder, title string, data *SeriesFields) {
	sb.WriteString(title + "\n\n")

	if len(data.MidPrices) > 0 {
		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidPrices)))
	}

	if len(data.EMA20Values) > 0 {
		sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.EMA20Values)))
	}

	if len(data.MACDValues) > 0 {
		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.MACDValues)))
	}

	if len(data.RSI7Values) > 0 {
		sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.RSI7Values)))
	}

	if len(data.RSI14Values) > 0 {
		sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.RSI14Values)))
	}

	if len(data.Volume) > 0 {
		sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.Volume)))
	}

	if len(data.ATR14Values) > 0 {
		sb.WriteString(fmt.Sprintf("ATR (14‑period): %s\n\n", formatFloatSlice(data.ATR14Values)))
	}

	if len(data.ER10Values) > 0 {
		sb.WriteString(fmt.Sprintf("Efficiency Ratio (10‑period): %s\n\n", formatFloatSlice(data.ER10Values)))
	}

	if len(data.BollingerPercentBs) > 0 {
		sb.WriteString(fmt.Sprintf("Bollinger %%B: %s\n\n", formatFloatSlice(data.BollingerPercentBs)))
	}

	if len(data.BollingerBandwidths) > 0 {
		sb.WriteString(fmt.Sprintf("Bollinger Bandwidth: %s\n\n", formatFloatSlice(data.BollingerBandwidths)))
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// BuildDataFromKlines 根据预加载的K线序列构造市场数据快照（用于回测/模拟）。
func BuildDataFromKlines(symbol string, primary []Kline, longer []Kline) (*Data, error) {
	if len(primary) == 0 {
		return nil, fmt.Errorf("primary series is empty")
	}

	symbol = Normalize(symbol)
	current := primary[len(primary)-1]
	currentPrice := current.Close

	data := &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		CurrentEMA20:      calculateEMA(primary, 20),
		CurrentMACD:       calculateMACD(primary),
		CurrentRSI7:       calculateRSI(primary, 7),
		PriceChange1h:     priceChangeFromSeries(primary, time.Hour),
		PriceChange4h:     priceChangeFromSeries(primary, 4*time.Hour),
		OpenInterest:      &OIData{Latest: 0, Average: 0},
		FundingRate:       0,
		IntradaySeries:    calculateIntradaySeries(primary),
		LongerTermContext: nil,
	}

	if len(longer) > 0 {
		data.LongerTermContext = calculateLongerTermData(longer)
	}

	return data, nil
}

func priceChangeFromSeries(series []Kline, duration time.Duration) float64 {
	if len(series) == 0 || duration <= 0 {
		return 0
	}
	last := series[len(series)-1]
	target := last.CloseTime - duration.Milliseconds()
	for i := len(series) - 1; i >= 0; i-- {
		if series[i].CloseTime <= target {
			price := series[i].Close
			if price > 0 {
				return ((last.Close - price) / price) * 100
			}
			break
		}
	}
	return 0
}

// isStaleData detects stale data (consecutive price freeze)
// Fix DOGEUSDT-style issue: consecutive N periods with completely unchanged prices indicate data source anomaly
func isStaleData(klines []Kline, symbol string) bool {
	if len(klines) < 2 {
		return false // Insufficient data to determine
	}

	// Detection threshold: 2 consecutive 5-minute periods with unchanged price (10 minutes without fluctuation)
	const stalePriceThreshold = 2
	const priceTolerancePct = 0.0001 // 0.01% fluctuation tolerance (avoid false positives)

	// Take the last stalePriceThreshold K-lines
	recentKlines := klines[len(klines)-stalePriceThreshold:]
	firstPrice := recentKlines[0].Close

	// Check if all prices are within tolerance
	for i := 1; i < len(recentKlines); i++ {
		priceDiff := math.Abs(recentKlines[i].Close-firstPrice) / firstPrice
		if priceDiff > priceTolerancePct {
			return false // Price fluctuation exists, data is normal
		}
	}

	// Additional check: MACD and volume
	// If price is unchanged but MACD/volume shows normal fluctuation, it might be a real market situation (extremely low volatility)
	// Check if volume is also 0 (data completely frozen)
	allVolumeZero := true
	for _, k := range recentKlines {
		if k.Volume > 0 {
			allVolumeZero = false
			break
		}
	}

	if allVolumeZero {
		log.Printf("⚠️  %s stale data confirmed: price freeze + zero volume", symbol)
		return true
	}

	// Price frozen but has volume: might be extremely low volatility market, allow but log warning
	log.Printf("⚠️  %s detected extreme price stability (no fluctuation for %d consecutive periods), but volume is normal", symbol, stalePriceThreshold)
	return false
}
