package mcp

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"nofx/logger"
)

// ModelMaxTokens defines official max output tokens for each model
var ModelMaxTokens = map[string]int{
	// Anthropic Claude
	"claude-opus-4-6":   128000,
	"claude-opus-4-5":   128000,
	"claude-opus-4":     128000,
	"claude-sonnet-4-6":  64000,
	"claude-sonnet-4-5":  64000,
	"claude-sonnet-4":    64000,

	// OpenAI
	"gpt-5.3-codex":     128000,
	"gpt-5.3":           128000,
	"gpt-5.2-codex":     64000,
	"gpt-5.2":           64000,
	"gpt-5.1-codex":     64000,
	"gpt-5.1":           64000,
	"gpt-5-codex":       64000,
	"gpt-5":             64000,

	// Google Gemini
	"gemini-3.1-pro-preview": 64000,
	"gemini-3-pro-preview":   64000,
	"gemini-3-flash-preview": 64000,

	// DeepSeek
	"deepseek-chat":      4096,
	"deepseek-coder":    4096,
	"deepseek-reasoner": 32000,

	// Qwen
	"qwen-turbo":         8192,
	"qwen-plus":         32768,
	"qwen-max":          32768,

	// Default fallback
	"default":            4096,
}

// ModelAliases maps third-party/model variant names to canonical model IDs
var ModelAliases = map[string]string{
	// Anthropic Claude aliases
	"opus":               "claude-opus-4",
	"opus-4":             "claude-opus-4",
	"claude-opus":        "claude-opus-4",
	"sonnet":             "claude-sonnet-4",
	"sonnet-4":           "claude-sonnet-4",
	"claude-sonnet":      "claude-sonnet-4",

	// OpenAI GPT aliases
	"gpt5":               "gpt-5",
	"gpt-5.0":           "gpt-5",
	"gpt5-codex":        "gpt-5-codex",
	"chatgpt-5":         "gpt-5",

	// Google Gemini aliases
	"gemini-3-pro":       "gemini-3-pro-preview",
	"gemini-3-flash":     "gemini-3-flash-preview",
	"gemini-3.1-pro":     "gemini-3.1-pro-preview",

	// DeepSeek aliases
	"deepseek":           "deepseek-chat",
	"deepseek-v3":       "deepseek-chat",

	// Qwen aliases
	"qwen":               "qwen-turbo",
	"qwen2.5":           "qwen-turbo",
	"qwen2":             "qwen-turbo",
}

// GetMaxTokensForModel returns the appropriate MaxTokens for the given model
// Priority: 1. Environment variable AI_MAX_TOKENS 2. Model-specific official limit 3. Default
func GetMaxTokensForModel(model string) int {
	// Check environment variable first (allows override)
	if envVal := os.Getenv("AI_MAX_TOKENS"); envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil && parsed > 0 {
			return parsed
		}
	}

	if model == "" {
		return ModelMaxTokens["default"]
	}

	modelLower := strings.ToLower(model)

	// Try exact match first
	if tokens, ok := ModelMaxTokens[modelLower]; ok {
		return tokens
	}

	// Try alias resolution
	if canonical, ok := ModelAliases[modelLower]; ok {
		if tokens, ok := ModelMaxTokens[canonical]; ok {
			return tokens
		}
	}

	// Try prefix match (e.g., "claude-opus-4-6" matches "claude-opus")
	for key, tokens := range ModelMaxTokens {
		if strings.HasPrefix(modelLower, key) || strings.HasPrefix(key, modelLower) {
			return tokens
		}
	}

	// Try stripping provider prefix (e.g., "anthropic/claude-sonnet-4-6" -> "claude-sonnet-4-6")
	modelName := stripProviderPrefix(modelLower)
	if modelName != modelLower {
		if tokens, ok := ModelMaxTokens[modelName]; ok {
			return tokens
		}
		if canonical, ok := ModelAliases[modelName]; ok {
			if tokens, ok := ModelMaxTokens[canonical]; ok {
				return tokens
			}
		}
		for key, tokens := range ModelMaxTokens {
			if strings.HasPrefix(modelName, key) || strings.HasPrefix(key, modelName) {
				return tokens
			}
		}
	}

	return ModelMaxTokens["default"]
}

// stripProviderPrefix removes provider prefix from model name
// e.g., "anthropic/claude-sonnet-4-6" -> "claude-sonnet-4-6"
// e.g., "openai/gpt-5" -> "gpt-5"
// e.g., "google/gemini-3-pro" -> "gemini-3-pro"
func stripProviderPrefix(model string) string {
	providers := []string{"anthropic/", "openai/", "google/", "deepseek/", "qwen/"}
	lower := strings.ToLower(model)
	for _, p := range providers {
		if strings.HasPrefix(lower, p) {
			return model[len(p):]
		}
	}
	return model
}

// Config client configuration (centralized management of all configurations)
type Config struct {
	// Provider configuration
	Provider     string
	APIKey       string
	BaseURL      string
	Model        string
	EndpointMode string

	// Behavior configuration
	MaxTokens   int
	Temperature float64
	UseFullURL  bool

	// Retry configuration
	MaxRetries      int
	RetryWaitBase   time.Duration
	RetryableErrors []string

	// Timeout configuration
	Timeout time.Duration

	// Dependency injection
	Logger     Logger
	HTTPClient *http.Client
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxTokens:       getEnvInt("AI_MAX_TOKENS", 0), // Will be set per-model in client initialization
		Temperature:     MCPClientTemperature,
		MaxRetries:      MaxRetryTimes,
		RetryWaitBase:   2 * time.Second,
		Timeout:         DefaultTimeout,
		RetryableErrors: retryableErrors,

		Logger:     logger.NewMCPLogger(),
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}
}

// getEnvInt reads integer from environment variable, returns default value if failed
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

// getEnvString reads string from environment variable, returns default value if empty
func getEnvString(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
