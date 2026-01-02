import { useState } from 'react'
import { Plus, X, Database, TrendingUp, List, Link, AlertCircle } from 'lucide-react'
import type { CoinSourceConfig } from '../../types'

// Default API URLs for data sources
const DEFAULT_COIN_POOL_API_URL = 'http://nofxaios.com:30006/api/ai500/list?auth=cm_568c67eae410d912c54c'
const DEFAULT_OI_TOP_API_URL = 'http://nofxaios.com:30006/api/oi/top-ranking?limit=20&duration=1h&auth=cm_568c67eae410d912c54c'

interface CoinSourceEditorProps {
  config: CoinSourceConfig
  onChange: (config: CoinSourceConfig) => void
  disabled?: boolean
  language: string
}

export function CoinSourceEditor({
  config,
  onChange,
  disabled,
  language,
}: CoinSourceEditorProps) {
  const [newCoin, setNewCoin] = useState('')

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      sourceType: { zh: '数据来源类型', en: 'Source Type' },
      static: { zh: '静态列表', en: 'Static List' },
      coinpool: { zh: 'AI500 数据源', en: 'AI500 Data Provider' },
      oi_top: { zh: 'OI Top 持仓增长', en: 'OI Top' },
      mixed: { zh: '混合模式', en: 'Mixed Mode' },
      staticCoins: { zh: '自定义币种', en: 'Custom Coins' },
      addCoin: { zh: '添加币种', en: 'Add Coin' },
      useCoinPool: { zh: '启用 AI500 数据源', en: 'Enable AI500 Data Provider' },
      coinPoolLimit: { zh: '数据源数量上限', en: 'Data Provider Limit' },
      coinPoolApiUrl: { zh: 'AI500 API URL', en: 'AI500 API URL' },
      coinPoolApiUrlPlaceholder: { zh: '输入 AI500 数据源 API 地址...', en: 'Enter AI500 data provider API URL...' },
      useOITop: { zh: '启用 OI Top 数据', en: 'Enable OI Top' },
      oiTopLimit: { zh: 'OI Top 数量上限', en: 'OI Top Limit' },
      oiTopApiUrl: { zh: 'OI Top API URL', en: 'OI Top API URL' },
      oiTopApiUrlPlaceholder: { zh: '输入 OI Top 持仓数据 API 地址...', en: 'Enter OI Top API URL...' },
      staticDesc: { zh: '手动指定交易币种列表', en: 'Manually specify trading coins' },
      coinpoolDesc: {
        zh: '使用 AI500 智能筛选的热门币种',
        en: 'Use AI500 smart-filtered popular coins',
      },
      oiTopDesc: {
        zh: '使用持仓量增长最快的币种',
        en: 'Use coins with fastest OI growth',
      },
      mixedDesc: {
        zh: '组合多种数据源，AI500 + OI Top + 自定义',
        en: 'Combine multiple sources: AI500 + OI Top + Custom',
      },
      apiUrlRequired: { zh: '需要填写 API URL 才能获取数据', en: 'API URL required to fetch data' },
      dataSourceConfig: { zh: '数据源配置', en: 'Data Source Configuration' },
      fillDefault: { zh: '填入默认', en: 'Fill Default' },
    }
    return translations[key]?.[language] || key
  }

  const sourceTypes = [
    { value: 'static', icon: List, color: '#848E9C' },
    { value: 'coinpool', icon: Database, color: '#F0B90B' },
    { value: 'oi_top', icon: TrendingUp, color: '#0ECB81' },
    { value: 'mixed', icon: Database, color: '#60a5fa' },
  ] as const

  // xyz dex assets (stocks, forex, commodities) - should NOT get USDT suffix
  const xyzDexAssets = new Set([
    // Stocks
    'TSLA', 'NVDA', 'AAPL', 'MSFT', 'META', 'AMZN', 'GOOGL', 'AMD', 'COIN', 'NFLX',
    'PLTR', 'HOOD', 'INTC', 'MSTR', 'TSM', 'ORCL', 'MU', 'RIVN', 'COST', 'LLY',
    'CRCL', 'SKHX', 'SNDK',
    // Forex
    'EUR', 'JPY',
    // Commodities
    'GOLD', 'SILVER',
    // Index
    'XYZ100',
  ])

  const isXyzDexAsset = (symbol: string): boolean => {
    const base = symbol.toUpperCase().replace(/^XYZ:/, '').replace(/USDT$|USD$|-USDC$/, '')
    return xyzDexAssets.has(base)
  }

  const handleAddCoin = () => {
    if (!newCoin.trim()) return
    const symbol = newCoin.toUpperCase().trim()

    // For xyz dex assets (stocks, forex, commodities), use xyz: prefix without USDT
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      // Remove xyz: prefix (case-insensitive) and any USD suffixes
      const base = symbol.replace(/^xyz:/i, '').replace(/USDT$|USD$|-USDC$/i, '')
      formattedSymbol = `xyz:${base}`
    } else {
      formattedSymbol = symbol.endsWith('USDT') ? symbol : `${symbol}USDT`
    }

    const currentCoins = config.static_coins || []
    if (!currentCoins.includes(formattedSymbol)) {
      onChange({
        ...config,
        static_coins: [...currentCoins, formattedSymbol],
      })
    }
    setNewCoin('')
  }

  const handleRemoveCoin = (coin: string) => {
    onChange({
      ...config,
      static_coins: (config.static_coins || []).filter((c) => c !== coin),
    })
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* Source Type Selector */}
      <div>
        <label className="block text-xs sm:text-sm font-medium mb-2 sm:mb-3" style={{ color: '#EAECEF' }}>
          {t('sourceType')}
        </label>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 sm:gap-3">
          {sourceTypes.map(({ value, icon: Icon, color }) => (
            <button
              key={value}
              onClick={() =>
                !disabled &&
                onChange({ ...config, source_type: value as CoinSourceConfig['source_type'] })
              }
              disabled={disabled}
              className={`p-2 sm:p-4 rounded-lg border transition-all ${
                config.source_type === value
                  ? 'ring-2 ring-yellow-500'
                  : 'hover:bg-white/5'
              }`}
              style={{
                background:
                  config.source_type === value
                    ? 'rgba(240, 185, 11, 0.1)'
                    : '#0B0E11',
                borderColor: '#2B3139',
              }}
            >
              <Icon className="w-4 h-4 sm:w-6 sm:h-6 mx-auto mb-1 sm:mb-2" style={{ color }} />
              <div className="text-xs sm:text-sm font-medium" style={{ color: '#EAECEF' }}>
                {t(value)}
              </div>
              <div className="text-[10px] sm:text-xs mt-0.5 sm:mt-1 line-clamp-2" style={{ color: '#848E9C' }}>
                {t(`${value}Desc`)}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Static Coins */}
      {(config.source_type === 'static' || config.source_type === 'mixed') && (
        <div>
          <label className="block text-xs sm:text-sm font-medium mb-2 sm:mb-3" style={{ color: '#EAECEF' }}>
            {t('staticCoins')}
          </label>
          <div className="flex flex-wrap gap-1.5 sm:gap-2 mb-2 sm:mb-3">
            {(config.static_coins || []).map((coin) => (
              <span
                key={coin}
                className="flex items-center gap-0.5 sm:gap-1 px-2 sm:px-3 py-1 sm:py-1.5 rounded-full text-xs sm:text-sm"
                style={{ background: '#2B3139', color: '#EAECEF' }}
              >
                <span className="truncate max-w-[120px] sm:max-w-none">{coin}</span>
                {!disabled && (
                  <button
                    onClick={() => handleRemoveCoin(coin)}
                    className="ml-0.5 sm:ml-1 hover:text-red-400 transition-colors flex-shrink-0"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </span>
            ))}
          </div>
          {!disabled && (
            <div className="flex flex-col sm:flex-row gap-2">
              <input
                type="text"
                value={newCoin}
                onChange={(e) => setNewCoin(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddCoin()}
                placeholder="BTC, ETH, SOL..."
                className="flex-1 px-2 sm:px-4 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm"
                style={{
                  background: '#0B0E11',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <button
                onClick={handleAddCoin}
                className="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg flex items-center justify-center gap-1.5 sm:gap-2 transition-colors text-xs sm:text-sm whitespace-nowrap"
                style={{ background: '#F0B90B', color: '#0B0E11' }}
              >
                <Plus className="w-3 h-3 sm:w-4 sm:h-4" />
                {t('addCoin')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Coin Pool Options */}
      {(config.source_type === 'coinpool' || config.source_type === 'mixed') && (
        <div className="space-y-3 sm:space-y-4">
          <div className="flex items-center gap-1.5 sm:gap-2 mb-1.5 sm:mb-2">
            <Link className="w-3 h-3 sm:w-4 sm:h-4 flex-shrink-0" style={{ color: '#F0B90B' }} />
            <span className="text-xs sm:text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('dataSourceConfig')} - AI500
            </span>
          </div>

          <div className="space-y-3">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 sm:gap-4">
              <label className="flex items-center gap-2 sm:gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={config.use_coin_pool}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, use_coin_pool: e.target.checked })
                  }
                  disabled={disabled}
                  className="w-4 h-4 sm:w-5 sm:h-5 rounded accent-yellow-500 flex-shrink-0"
                />
                <span className="text-xs sm:text-sm" style={{ color: '#EAECEF' }}>{t('useCoinPool')}</span>
              </label>
              {config.use_coin_pool && (
                <div className="flex items-center gap-2 sm:gap-3">
                  <span className="text-xs sm:text-sm whitespace-nowrap" style={{ color: '#848E9C' }}>
                    {t('coinPoolLimit')}:
                  </span>
                  <input
                    type="number"
                    value={config.coin_pool_limit || 10}
                    onChange={(e) =>
                      !disabled &&
                      onChange({ ...config, coin_pool_limit: parseInt(e.target.value) || 10 })
                    }
                    disabled={disabled}
                    min={1}
                    max={100}
                    className="w-20 sm:w-24 px-2 sm:px-3 py-1.5 sm:py-2 rounded text-xs sm:text-sm"
                    style={{
                      background: '#0B0E11',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                    }}
                  />
                </div>
              )}
            </div>

            {config.use_coin_pool && (
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <label className="text-xs sm:text-sm" style={{ color: '#848E9C' }}>
                    {t('coinPoolApiUrl')}
                  </label>
                  {!disabled && !config.coin_pool_api_url && (
                    <button
                      type="button"
                      onClick={() => onChange({ ...config, coin_pool_api_url: DEFAULT_COIN_POOL_API_URL })}
                      className="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0.5 sm:py-1 rounded flex-shrink-0"
                      style={{ background: '#F0B90B20', color: '#F0B90B' }}
                    >
                      {t('fillDefault')}
                    </button>
                  )}
                </div>
                <input
                  type="url"
                  value={config.coin_pool_api_url || ''}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, coin_pool_api_url: e.target.value })
                  }
                  disabled={disabled}
                  placeholder={t('coinPoolApiUrlPlaceholder')}
                  className="w-full px-2 sm:px-4 py-1.5 sm:py-2.5 rounded-lg font-mono text-xs sm:text-sm"
                  style={{
                    background: '#0B0E11',
                    border: '1px solid #2B3139',
                    color: '#EAECEF',
                  }}
                />
                {!config.coin_pool_api_url && (
                  <div className="flex items-center gap-1.5 sm:gap-2 mt-1.5 sm:mt-2">
                    <AlertCircle className="w-3 h-3 sm:w-4 sm:h-4 flex-shrink-0" style={{ color: '#F0B90B' }} />
                    <span className="text-[10px] sm:text-xs" style={{ color: '#F0B90B' }}>
                      {t('apiUrlRequired')}
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* OI Top Options */}
      {(config.source_type === 'oi_top' || config.source_type === 'mixed') && (
        <div className="space-y-3 sm:space-y-4">
          <div className="flex items-center gap-1.5 sm:gap-2 mb-1.5 sm:mb-2">
            <Link className="w-3 h-3 sm:w-4 sm:h-4 flex-shrink-0" style={{ color: '#0ECB81' }} />
            <span className="text-xs sm:text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('dataSourceConfig')} - OI Top
            </span>
          </div>

          <div className="space-y-3">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 sm:gap-4">
              <label className="flex items-center gap-2 sm:gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={config.use_oi_top}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, use_oi_top: e.target.checked })
                  }
                  disabled={disabled}
                  className="w-4 h-4 sm:w-5 sm:h-5 rounded accent-yellow-500 flex-shrink-0"
                />
                <span className="text-xs sm:text-sm" style={{ color: '#EAECEF' }}>{t('useOITop')}</span>
              </label>
              {config.use_oi_top && (
                <div className="flex items-center gap-2 sm:gap-3">
                  <span className="text-xs sm:text-sm whitespace-nowrap" style={{ color: '#848E9C' }}>
                    {t('oiTopLimit')}:
                  </span>
                  <input
                    type="number"
                    value={config.oi_top_limit || 20}
                    onChange={(e) =>
                      !disabled &&
                      onChange({ ...config, oi_top_limit: parseInt(e.target.value) || 20 })
                    }
                    disabled={disabled}
                    min={1}
                    max={50}
                    className="w-20 sm:w-24 px-2 sm:px-3 py-1.5 sm:py-2 rounded text-xs sm:text-sm"
                    style={{
                      background: '#0B0E11',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                    }}
                  />
                </div>
              )}
            </div>

            {config.use_oi_top && (
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <label className="text-xs sm:text-sm" style={{ color: '#848E9C' }}>
                    {t('oiTopApiUrl')}
                  </label>
                  {!disabled && !config.oi_top_api_url && (
                    <button
                      type="button"
                      onClick={() => onChange({ ...config, oi_top_api_url: DEFAULT_OI_TOP_API_URL })}
                      className="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0.5 sm:py-1 rounded flex-shrink-0"
                      style={{ background: '#0ECB8120', color: '#0ECB81' }}
                    >
                      {t('fillDefault')}
                    </button>
                  )}
                </div>
                <input
                  type="url"
                  value={config.oi_top_api_url || ''}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, oi_top_api_url: e.target.value })
                  }
                  disabled={disabled}
                  placeholder={t('oiTopApiUrlPlaceholder')}
                  className="w-full px-2 sm:px-4 py-1.5 sm:py-2.5 rounded-lg font-mono text-xs sm:text-sm"
                  style={{
                    background: '#0B0E11',
                    border: '1px solid #2B3139',
                    color: '#EAECEF',
                  }}
                />
                {!config.oi_top_api_url && (
                  <div className="flex items-center gap-1.5 sm:gap-2 mt-1.5 sm:mt-2">
                    <AlertCircle className="w-3 h-3 sm:w-4 sm:h-4 flex-shrink-0" style={{ color: '#F0B90B' }} />
                    <span className="text-[10px] sm:text-xs" style={{ color: '#F0B90B' }}>
                      {t('apiUrlRequired')}
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
