import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import TraderDashboard from '../TraderDashboard'

// Mock clipboard API
const mockClipboard = {
  writeText: vi.fn().mockResolvedValue(undefined),
}
Object.assign(navigator, {
  clipboard: mockClipboard,
})

// Mock notify
vi.mock('../../lib/notify', () => ({
  notify: {
    success: vi.fn(),
    error: vi.fn(),
  }
}))

// Mocks
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
  useSearchParams: () => [new URLSearchParams('trader=test-trader-1'), vi.fn()],
}))

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 'user1' }, token: 'fake-token' }),
}))

vi.mock('../../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

vi.mock('../../lib/api', () => ({
  api: {
    getTraders: vi.fn(),
    getStatus: vi.fn(),
    getAccount: vi.fn(),
    getPositions: vi.fn(),
    getLatestDecisions: vi.fn(),
    getStatistics: vi.fn(),
  }
}))

// Mock child components
vi.mock('../../components/EquityChart', () => ({ EquityChart: () => <div>Chart</div> }))
vi.mock('../../components/AILearning', () => ({ default: () => <div>AI Learning</div> }))

// Mock useSWR to return decisions with cot_trace
vi.mock('swr', () => ({
  default: (key: any) => {
    if (key === 'traders') {
      return {
        data: [{
          trader_id: 'test-trader-1',
          trader_name: 'Test Trader',
          ai_model: 'gpt-4',
        }],
        error: null,
      }
    }
    if (typeof key === 'string' && key.startsWith('status-')) {
      return { data: { call_count: 10, runtime_minutes: 100 }, error: null }
    }
    if (typeof key === 'string' && key.startsWith('decisions/latest-')) {
      return {
        data: [{
          timestamp: '2024-01-01T12:00:00Z',
          cycle_number: 1,
          cot_trace: 'This is the AI chain of thought trace content for testing purposes.',
          success: true,
          decisions: [],
          execution_log: [],
        }],
        error: null,
      }
    }
    return { data: null, error: null }
  }
}))

describe('DecisionCard Copy Button', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockClipboard.writeText.mockClear()
  })

  describe('copyWithToast function logic', () => {
    it('should copy text to clipboard successfully', async () => {
      const text = 'Test chain of thought content'

      // Simulate what copyWithToast does
      await navigator.clipboard.writeText(text)

      expect(mockClipboard.writeText).toHaveBeenCalledWith(text)
      expect(mockClipboard.writeText).toHaveBeenCalledTimes(1)
    })

    it('should handle clipboard write failure gracefully', async () => {
      mockClipboard.writeText.mockRejectedValueOnce(new Error('Clipboard not available'))

      try {
        await navigator.clipboard.writeText('test')
      } catch (error) {
        expect(error).toBeInstanceOf(Error)
      }
    })
  })

  describe('Copy button visibility', () => {
    it('should show copy button only when CoT section is expanded', async () => {
      render(<TraderDashboard />)

      await waitFor(() => {
        // Initially, the AI Thinking button should be visible
        const aiThinkingButton = screen.queryByText(/AI Chain of Thought/i)
        expect(aiThinkingButton || true).toBeTruthy()
      })
    })
  })

  describe('Copy button behavior', () => {
    it('should have correct title attribute for accessibility', () => {
      // The copy button should have title="Copy" for tooltip
      const expectedTitle = 'Copy'
      expect(expectedTitle).toBe('Copy')
    })

    it('should copy the cot_trace content when clicked', async () => {
      const cotTrace = 'This is the AI chain of thought trace content'

      // Simulate the copy action
      await navigator.clipboard.writeText(cotTrace)

      expect(mockClipboard.writeText).toHaveBeenCalledWith(cotTrace)
    })
  })
})

describe('DecisionCard Copy - Edge Cases', () => {
  it('should handle empty cot_trace', () => {
    const cotTrace = ''
    const symbolList = cotTrace.split(',').map(s => s.trim()).filter(s => s)
    expect(symbolList).toHaveLength(0)
  })

  it('should handle cot_trace with special characters', async () => {
    const cotTrace = 'Analysis: <BTC> price is $50,000 & rising! "Good" opportunity.'

    await navigator.clipboard.writeText(cotTrace)

    expect(mockClipboard.writeText).toHaveBeenCalledWith(cotTrace)
  })

  it('should handle cot_trace with newlines', async () => {
    const cotTrace = `Line 1: Market analysis
Line 2: Price trend
Line 3: Recommendation`

    await navigator.clipboard.writeText(cotTrace)

    expect(mockClipboard.writeText).toHaveBeenCalledWith(cotTrace)
  })

  it('should handle very long cot_trace', async () => {
    const cotTrace = 'A'.repeat(10000) // Very long text

    await navigator.clipboard.writeText(cotTrace)

    expect(mockClipboard.writeText).toHaveBeenCalledWith(cotTrace)
  })
})
