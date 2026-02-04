package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		c.detectedMode = EndpointModeNative
		c.logger.Infof("ℹ️  [MCP] Detected Google official endpoint (native)")
		return
	}

	detected := c.probeEndpoint()
	c.detectedMode = detected

	switch detected {
	case EndpointModeNative:
		c.logger.Infof("✅ [MCP] Detected native Gemini endpoint (context caching supported)")
	case EndpointModeCompatible:
		c.logger.Infof("✅ [MCP] Detected OpenAI compatible endpoint")
	default:
		c.detectedMode = EndpointModeCompatible
		c.logger.Infof("⚠️  [MCP] Could not detect endpoint type, defaulting to OpenAI compatible mode")
	}
}

func (c *GeminiClient) probeEndpoint() EndpointMode {
	// Try native endpoint first
	if c.testNativeEndpoint() {
		return EndpointModeNative
	}
	// Fall back to compatible endpoint
	if c.testCompatibleEndpoint() {
		return EndpointModeCompatible
	}
	return EndpointModeCompatible
}

func (c *GeminiClient) testCompatibleEndpoint() bool {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")

	var testURL string
	if strings.HasSuffix(baseURL, "/chat/completions") {
		testURL = baseURL
	} else if strings.HasSuffix(baseURL, "/v1") {
		testURL = baseURL + "/chat/completions"
	} else {
		testURL = baseURL + "/chat/completions"
	}

	c.logger.Debugf("🔍 [MCP] Testing compatible endpoint: %s", testURL)

	testBody := map[string]any{
		"model":       c.Model,
		"max_tokens":  1,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	jsonData, _ := json.Marshal(testBody)
	client := &http.Client{Timeout: 15 * time.Second}

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			c.logger.Debugf("🔄 [MCP] Retrying compatible endpoint test (attempt %d/3)", attempt+1)
		}

		req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
		if err != nil {
			c.logger.Debugf("❌ [MCP] Failed to create compatible endpoint request: %v", err)
			return false
		}

		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("content-type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.logger.Debugf("❌ [MCP] Compatible endpoint request failed: %v", err)
			if attempt < 2 {
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			c.logger.Debugf("✅ [MCP] Compatible endpoint detected")
			return true
		}

		c.logger.Debugf("❌ [MCP] Compatible endpoint returned status: %d", resp.StatusCode)
		if attempt < 2 {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}

	return false
}

func (c *GeminiClient) testNativeEndpoint() bool {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")

	var testURL string
	if strings.Contains(baseURL, ":generateContent") {
		testURL = baseURL
	} else if strings.HasSuffix(baseURL, "/v1beta") {
		testURL = fmt.Sprintf("%s/models/%s:generateContent", baseURL, c.Model)
	} else {
		testURL = fmt.Sprintf("%s/models/%s:generateContent", baseURL, c.Model)
	}

	c.logger.Debugf("🔍 [MCP] Testing native endpoint: %s", testURL)

	testBody := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": "hi"},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": 1,
			"temperature":     0,
		},
	}

	jsonData, _ := json.Marshal(testBody)
	client := &http.Client{Timeout: 15 * time.Second}

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			c.logger.Debugf("🔄 [MCP] Retrying native endpoint test (attempt %d/3)", attempt+1)
		}

		req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
		if err != nil {
			c.logger.Debugf("❌ [MCP] Failed to create native endpoint request: %v", err)
			return false
		}

		req.Header.Set("x-goog-api-key", c.APIKey)
		req.Header.Set("content-type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.logger.Debugf("❌ [MCP] Native endpoint request failed: %v", err)
			if attempt < 2 {
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.logger.Debugf("❌ [MCP] Native endpoint returned status: %d", resp.StatusCode)
			if attempt < 2 {
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			return false
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			c.logger.Debugf("❌ [MCP] Failed to parse native endpoint response: %v", err)
			if attempt < 2 {
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			return false
		}

		_, hasCandidates := result["candidates"]
		if hasCandidates {
			c.logger.Debugf("✅ [MCP] Native endpoint detected with candidates field")
			return true
		}

		return false
	}

	return false
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
	mode := c.GetEndpointMode()

	if mode == EndpointModeNative {
		reqHeaders.Set("x-goog-api-key", c.APIKey)
	} else {
		c.Client.setAuthHeader(reqHeaders)
	}
}

func (c *GeminiClient) buildUrl() string {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	mode := c.GetEndpointMode()

	if mode == EndpointModeNative {
		if strings.Contains(baseURL, ":generateContent") {
			return baseURL
		}
		if strings.HasSuffix(baseURL, "/v1beta") {
			return fmt.Sprintf("%s/models/%s:generateContent", baseURL, c.Model)
		}
		return fmt.Sprintf("%s/v1beta/models/%s:generateContent", baseURL, c.Model)
	}

	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func (c *GeminiClient) buildMCPRequestBody(systemPrompt, userPrompt string) map[string]any {
	mode := c.GetEndpointMode()

	if mode == EndpointModeNative {
		return c.buildNativeRequest(systemPrompt, userPrompt)
	}
	return c.buildCompatibleRequest(systemPrompt, userPrompt)
}

func (c *GeminiClient) buildNativeRequest(systemPrompt, userPrompt string) map[string]any {
	parts := []map[string]string{
		{"text": userPrompt},
	}

	if systemPrompt != "" {
		parts = append([]map[string]string{{"text": systemPrompt}}, parts...)
	}

	requestBody := map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": parts,
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": c.MaxTokens,
		},
	}

	if c.config.Temperature > 0 {
		requestBody["generationConfig"].(map[string]any)["temperature"] = c.config.Temperature
	}

	return requestBody
}

func (c *GeminiClient) buildCompatibleRequest(systemPrompt, userPrompt string) map[string]any {
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
	mode := c.GetEndpointMode()

	if mode == EndpointModeNative {
		return c.parseNativeResponse(body)
	}
	return c.parseCompatibleResponse(body)
}

func (c *GeminiClient) parseNativeResponse(body []byte) (string, error) {
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata *struct {
			PromptTokenCount        int `json:"promptTokenCount"`
			CandidatesTokenCount    int `json:"candidatesTokenCount"`
			CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
		} `json:"usageMetadata"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse Gemini native response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("Gemini API error: %s", response.Error.Message)
	}

	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in Gemini response")
	}

	if len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini response")
	}

	if response.UsageMetadata != nil && response.UsageMetadata.CachedContentTokenCount > 0 {
		c.logger.Infof("💰 [MCP] Context cache hit! %d tokens from cache", response.UsageMetadata.CachedContentTokenCount)
	}

	if response.UsageMetadata != nil && TokenUsageCallback != nil {
		totalTokens := response.UsageMetadata.PromptTokenCount + response.UsageMetadata.CandidatesTokenCount
		if totalTokens > 0 {
			TokenUsageCallback(TokenUsage{
				Provider:         c.Provider,
				Model:            c.Model,
				PromptTokens:     response.UsageMetadata.PromptTokenCount,
				CompletionTokens: response.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      totalTokens,
			})
		}
	}

	return response.Candidates[0].Content.Parts[0].Text, nil
}

func (c *GeminiClient) parseCompatibleResponse(body []byte) (string, error) {
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
		return "", fmt.Errorf("failed to parse Gemini compatible response: %w", err)
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
