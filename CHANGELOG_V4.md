# z-dev-v4 Changelog

## 版本資訊

- **分支名稱**: `z-dev-v4`
- **基於上游**: `nofxaios/nofx` @ commit `4a0f56f1` (2024-12-02)
- **創建日期**: 2024-12-08
- **狀態**: ✅ 穩定版本，建議作為 Fork 默認分支

## 核心理念

z-dev-v4 採用 **「最小修改，最大穩定」** 策略：
- ✅ **完全同步上游最新代碼**（0 commits 差距）
- ✅ **僅添加必要的關鍵修復**（5 commits）
- ✅ **不引入實驗性功能**
- ✅ **保持與上游 100% 兼容**

## 與 V2/V3 的差異

| 特性 | z-dev-v2 | z-dev-v3 | z-dev-v4 ✨ |
|------|----------|----------|-------------|
| **上游同步狀態** | ~769 commits 落後 | ~769 commits 落後 | ✅ **完全同步** |
| **K線週期問題** | ❌ 未解決 | ❌ 未解決 | ✅ **90% 解決** (上游修復) |
| **AI 數據幻覺** | ❌ 未解決 | ❌ 未解決 | ✅ **新增約束** |
| **啟動卡住問題** | ❌ 存在 | ❌ 存在 | ✅ **已解決** (上游修復) |
| **AutoStart 功能** | ❌ 未啟用 | ❌ 未啟用 | ✅ **已啟用** |
| **獨特功能數量** | 多個實驗性功能 | V2 + Bybit 修復 | **5 個關鍵修復** |
| **穩定性評估** | ⚠️ 不確定 | ⚠️ 不確定 | ✅ **最穩定** |

## 上游帶來的重大改進

z-dev-v4 繼承了上游的所有最新功能：

### ✅ **Strategy Studio（多時間框架支持）**
- 支持用戶自定義 K 線週期（5m, 15m, 1h, 4h 等）
- `GetWithTimeframes()` 函數提供多時間框架數據
- 解決了 90% 的「K線週期錯誤」問題

### ✅ **數據庫重構（store package）**
- SQLite 從 WAL 模式改為 DELETE 模式
- 完美解決 Docker 啟動卡住問題
- 提升跨平台兼容性

### ✅ **OKX 交易所支持**
- 完整的 OKX API 整合
- 支持 USDT/USDC/USD 結算類型

### ✅ **交易動作簡化**
- 從複雜的 10+ 動作簡化為 6 個核心動作
- 減少 AI 決策錯誤率

### ✅ **健康檢查優化**
- Docker healthcheck 啟動時間延長到 60s
- 避免誤判為不健康狀態

## z-dev-v4 獨特修復（5 commits）

### 1️⃣ **Commit 30d0bbc6**: OKX 類型安全修復
```
fix(okx): use safe type assertions to prevent panic in CloseLong/CloseShort
```
**問題**: OKX trader 在關閉倉位時使用不安全的類型斷言
**影響**: 可能導致 panic 崩潰
**修復**: 使用 `value, ok := data.(type)` 安全模式，添加錯誤處理

**文件變動**:
- `trader/okx_trader.go`: CloseLong/CloseShort 函數

---

### 2️⃣ **Commit 78e86c1f**: AI 數據約束（防止幻覺）
```
fix(ai): add critical data constraints to prevent hallucination
```
**問題**: AI 會虛構 RSI 值，引用不存在的歷史數據
**影響**: 交易決策基於錯誤數據，導致虧損
**修復**: 在系統提示詞中添加 5 條嚴格約束規則

**新增規則**:
1. ⚠️ **只能使用 User Prompt 中的確切數據**
2. ⚠️ **禁止虛構或估算任何指標值**
3. ⚠️ **數據缺失時必須輸出 "wait" 動作**
4. ⚠️ **禁止發明歷史背景**
5. ⚠️ **不確定時必須聲明 "INSUFFICIENT DATA"**

**文件變動**:
- `decision/engine.go`: 第 535-543 行（系統提示詞部分）

---

### 3️⃣ **Commit 0c4e38fe**: 啟用 AutoStartRunningTraders
```
feat(startup): enable auto-start for running traders
```
**問題**: 系統重啟後，之前運行中的 traders 不會自動恢復
**影響**: 用戶需要手動重新啟動每個 trader
**修復**: 在 main.go 啟動時調用 AutoStartRunningTraders()

**文件變動**:
- `main.go`: 第 123 行

---

### 4️⃣ **Commit e0c9ec74**: Fork 分支標籤顯示
```
feat(ui): add fork attribution labels (z-dev-v4)
```
**目的**: 明確標示這是實驗性社區版本
**內容**: 在 UI 底部顯示分支名稱和上游鏈接
**支持**: 雙語（中文/英文）

**文件變動**:
- `web/src/pages/AITradersPage.tsx`: Footer 部分
- `web/src/components/landing/FooterSection.tsx`: Footer 部分

**顯示內容**:
```
實驗性社區版本（非官方）
維護者：the-dev-z/nofx (z-dev-v4) | 上游：nofxaios/nofx
```

---

### 5️⃣ **Commit 2ca3043f**: 澄清 market.Get() 使用範圍
```
docs: clarify market.Get() usage scope with comments
```
**背景**: `market.Get()` 函數內部硬編碼使用 3m K線
**誤解**: 可能被誤認為影響 AI 決策的技術指標
**澄清**: 添加註釋說明 Get() 僅用於獲取價格/OI，不用於 AI 分析

**註釋位置** (5 處):
- `decision/engine.go:397` - OI 過濾
- `trader/auto_trader.go:750,833` - 計算下單數量
- `trader/auto_trader.go:906,945` - 記錄平倉價格

**重要說明**:
> AI 決策使用的是 `GetWithTimeframes()`，會使用用戶配置的時間框架（15m/1h/4h 等），不受 `Get()` 的 3m 硬編碼影響。

## 推薦使用場景

✅ **推薦使用 z-dev-v4 如果您需要**:
- 最新的上游功能和修復
- 最穩定的運行環境
- 多時間框架 K 線支持
- OKX 交易所支持
- 自動重啟 traders 功能
- AI 決策數據準確性保障

⚠️ **考慮 V2/V3 如果您需要**:
- 特定的實驗性功能
- 與舊版本保持完全一致

## 升級指南

### 從 z-dev-v2/v3 升級到 z-dev-v4:

1. **備份數據**:
   ```bash
   cp data.db data.db.backup_$(date +%Y%m%d)
   ```

2. **切換分支**:
   ```bash
   git fetch origin
   git checkout z-dev-v4
   ```

3. **重建前端** (如果需要):
   ```bash
   cd web
   npm install
   npm run build
   cd ..
   ```

4. **重啟系統**:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

5. **驗證**:
   - 檢查 traders 是否自動重啟
   - 查看 UI 底部是否顯示 "z-dev-v4" 標籤
   - 測試 AI 決策是否使用正確的 K 線週期

## 已知問題與限制

### ⚠️ 部分解決的問題

**K線週期硬編碼 (market.Get())**:
- **現狀**: `market.Get()` 函數內部仍使用 3m K線
- **影響**: 僅影響 OI 過濾和價格獲取（不影響 AI 決策）
- **說明**: AI 決策使用 `GetWithTimeframes()`，不受影響
- **未來**: 可能創建 `GetBasicData()` 輕量級函數

## 維護者

- **Fork 維護者**: [the-dev-z](https://github.com/the-dev-z)
- **上游項目**: [nofxaios/nofx](https://github.com/nofxaios/nofx)

## 授權

遵循上游項目的授權協議。

## 更新日誌

### 2024-12-08 - v4 首次發布
- ✅ 創建 z-dev-v4 分支
- ✅ 基於 upstream/dev 最新代碼
- ✅ 添加 5 個關鍵修復
- ✅ 完成文檔編寫
- ✅ 推送到 GitHub

---

**建議**: 將 z-dev-v4 設為 Fork 的默認分支，享受最穩定和最新的功能！
