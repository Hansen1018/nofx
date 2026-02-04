package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ProviderGemini       = "gemini"
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

	DefaultGeminiModel = "gemini-3-pro-preview"
	Gemini3Pro         = "gemini-3-pro-preview"
	Gemini3Flash       = "gemini-3-flash-preview"
	Gemini3Nano        = "gemini-3-nano-preview"

	Gemini25Pro      = "gemini-2.5-pro-preview"
	Gemini25Flash    = "gemini-2.5-flash-preview"
	Gemini25ProAlias = "gemini-2.5-pro-preview"
)

var geminiCachingModels = []string{
	"gemini-3-pro",
	"gemini-3-flash",
	"gemini-3-nano",
	"gemini-2.5-pro",
	"gemini-2.5-flash",
}

type GeminiClient struct {
	*Client
	endpointMode EndpointMode
	detectedMode EndpointMode
	cache        *geminiCache
}

type geminiCache struct {
	cacheName        string
	cacheCreateTime  time.Time
	systemPromptHash string
}

func NewGeminiClient() AIClient {
	return NewGeminiClientWithOptions()
}

func NewGeminiClientWithOptions(opts ...ClientOption) AIClient {
	geminiOpts := []ClientOption{
		WithProvider(ProviderGemini),
		WithModel(DefaultGeminiModel),
		WithBaseURL(DefaultGeminiBaseURL),
		WithEndpointMode(EndpointModeAuto),
	}

	allOpts := append(geminiOpts, opts...)
	baseClient := NewClient(allOpts...).(*Client)

	geminiClient := &GeminiClient{
		Client:       baseClient,
		endpointMode: EndpointModeAuto,
		cache:        &geminiCache{},
	}

	if baseClient.config.EndpointMode != "" {
		geminiClient.endpointMode = EndpointMode(baseClient.config.EndpointMode)
	}

	baseClient.hooks = geminiClient

	return geminiClient
}

func (c *GeminiClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	c.APIKey = apiKey

	if len(apiKey) > 8 {
		c.logger.Infof("🔧 [MCP] Gemini API Key: %s...%s", apiKey[:4], apiKey[len(apiKey)-4:])
	}

	if customURL != "" {
		c.BaseURL = customURL
		c.logger.Infof("🔧 [MCP] Gemini using custom BaseURL: %s", customURL)

		if c.endpointMode == EndpointModeAuto {
			c.detectEndpointMode()
		}
	} else {
		c.BaseURL = DefaultGeminiBaseURL
		c.detectedMode = EndpointModeCompatible
		c.logger.Infof("🔧 [MCP] Gemini using default OpenAI-compatible endpoint")
	}

	if customModel != "" {
		c.Model = customModel
		c.logger.Infof("🔧 [MCP] Gemini using custom Model: %s", customModel)
	} else {
		c.logger.Infof("🔧 [MCP] Gemini using default Model: %s", c.Model)
	}

	mode := c.endpointMode
	if mode == EndpointModeAuto {
		mode = c.detectedMode
	}
	c.logger.Infof("🎯 [MCP] Gemini endpoint mode: %s", mode)
}

func (c *GeminiClient) detectEndpointMode() {
	c.logger.Infof("🔍 [MCP] Detecting endpoint mode for: %s", c.BaseURL)

	urlLower := strings.ToLower(c.BaseURL)
	if strings.Contains(urlLower, "googleapis.com") ||
		strings.Contains(urlLower, "generativelanguage") {
		c.detectedMode = EndpointModeCompatible
		c.logger.Infof("ℹ️  [MCP] Detected Google official endpoint (OpenAI compatible)")
		return
	}

	detected := c.probeEndpoint()
	c.detectedMode = detected

	switch detected {
	case EndpointModeCompatible:
		c.logger.Infof("✅ [MCP] Detected OpenAI compatible endpoint")
	default:
		c.detectedMode = EndpointModeCompatible
		c.logger.Infof("⚠️  [MCP] Could not detect endpoint type, defaulting to OpenAI compatible mode")
	}
}

func (c *GeminiClient) probeEndpoint() EndpointMode {
	if c.testCompatibleEndpoint() {
		return EndpointModeCompatible
	}
	return EndpointModeCompatible
}

func (c *GeminiClient) testCompatibleEndpoint() bool {
	testURL := fmt.Sprintf("%s/chat/completions", c.BaseURL)

	testBody := map[string]any{
		"model":       c.Model,
		"max_tokens":  1,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	jsonData, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (c *GeminiClient) GetEndpointMode() EndpointMode {
	if c.endpointMode == EndpointModeAuto {
		return c.detectedMode
	}
	return c.endpointMode
}

func (c *GeminiClient) IsContextCachingEnabled() bool {
	return c.supportsContextCaching()
}

func (c *GeminiClient) supportsContextCaching() bool {
	modelLower := strings.ToLower(c.Model)

	for _, supported := range geminiCachingModels {
		if strings.Contains(modelLower, supported) {
			return true
		}
	}
	return false
}

func (c *GeminiClient) setAuthHeader(reqHeaders http.Header) {
	c.Client.setAuthHeader(reqHeaders)
}

func (c *GeminiClient) buildUrl() string {
	return fmt.Sprintf("%s/chat/completions", c.BaseURL)
}

func (c *GeminiClient) buildMCPRequestBody(systemPrompt, userPrompt string) map[string]any {
	messages := []map[string]string{}

	if systemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": systemPrompt,
		})
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userPrompt,
	})

	requestBody := map[string]any{
		"model":      c.Model,
		"messages":   messages,
		"max_tokens": c.MaxTokens,
	}

	if c.config.Temperature > 0 {
		requestBody["temperature"] = c.config.Temperature
	}

	return requestBody
}

func (c *GeminiClient) parseMCPResponse(body []byte) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			CachedTokens     int `json:"cached_tokens,omitempty"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("Gemini API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in Gemini response")
	}

	if response.Usage.CachedTokens > 0 {
		c.logger.Infof("💰 [MCP] Context cache hit! %d tokens from cache", response.Usage.CachedTokens)
	}

	totalTokens := response.Usage.PromptTokens + response.Usage.CompletionTokens
	if TokenUsageCallback != nil && totalTokens > 0 {
		TokenUsageCallback(TokenUsage{
			Provider:         c.Provider,
			Model:            c.Model,
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      totalTokens,
		})
	}

	return response.Choices[0].Message.Content, nil
}
