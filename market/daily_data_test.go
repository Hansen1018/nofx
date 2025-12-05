package market

import (
	"testing"
	"time"
)

// TestGetDailyData_MACDValues 验证日线数据的 MACD 计算
func TestGetDailyData_MACDValues(t *testing.T) {
	// 1. 保存原始的全局变量，并在测试结束时恢复
	originalWSMonitor := WSMonitorCli
	defer func() {
		WSMonitorCli = originalWSMonitor
	}()

	// 2. 初始化一个新的 WSMonitor 用于测试
	WSMonitorCli = &WSMonitor{}
	
	// 3. 生成足够的测试 K线数据 (至少需要 26+ 个点来计算 MACD)
	// 我们生成 50 个点
	klines := generateTestKlines(50)
	
	// 4. 将数据注入到 WSMonitor 的缓存中
	entry := &KlineCacheEntry{
		Klines:     klines,
		ReceivedAt: time.Now(),
	}
	WSMonitorCli.klineDataMap1d.Store("BTCUSDT", entry)
	
	// 5. 调用被测函数
	data, err := getDailyData("BTCUSDT")
	if err != nil {
		t.Fatalf("getDailyData failed: %v", err)
	}
	
	// 6. 验证 MACDValues
	if data.MACDValues == nil {
		t.Fatal("MACDValues is nil")
	}
	
	// DailyData 只保留最后 DailyDataPoints (7) 个点
	// 但指标计算是基于 fullKlines 的
	// getDailyData 内部会对 MACDValues 进行切片或者重新分配
	// 让我们检查一下 data.MACDValues 的长度
	if len(data.MACDValues) != len(data.ClosePrices) {
		t.Errorf("MACDValues length (%d) should match ClosePrices length (%d)", 
			len(data.MACDValues), len(data.ClosePrices))
	}
	
	// 检查最后几个 MACD 值是否非零 (假设 generateTestKlines 生成的数据有波动)
	// 注意：前 25 个点的 MACD 应该是 0 (或者无效值，取决于实现，这里 calculateMACD 返回 0 如果长度不够)
	// 但对于最后 7 个点，因为我们总共有 50 个点，所以它们应该都有有效的 MACD 值
	for i, macd := range data.MACDValues {
		// 对应的 fullKlines 索引
		// fullKlines 长度 50
		// data 长度 7 (DailyDataPoints)
		// startIdx = 50 - 7 = 43
		// fullIdx = 43 + i
		// 因为 fullIdx >= 43 >> 25，所以应该有值
		
		if macd == 0 && i > 0 { // 允许第一个偶尔为0，但不应该全是0
			// 注意：如果完全没有波动，MACD 可能是 0。但在 generateTestKlines 中有 variance。
			// 让我们打印一下值看看
			t.Logf("MACD[%d] = %f", i, macd)
		}
	}
	
	// 我们可以手动验证最后一个点的 MACD
	// 取 fullKlines
	lastMACD := calculateMACD(klines)
	if data.MACDValues[len(data.MACDValues)-1] != lastMACD {
		t.Errorf("Last MACD value mismatch. Got %f, want %f", 
			data.MACDValues[len(data.MACDValues)-1], lastMACD)
	}
}
