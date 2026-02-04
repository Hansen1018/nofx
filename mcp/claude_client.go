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

// EndpointMode 定义端点工作模式
type EndpointMode string

const (
	// EndpointModeAuto 自动检测模式（推荐）
	EndpointModeAuto EndpointMode = "auto"
	// EndpointModeCompatible OpenAI 兼容模式
	EndpointModeCompatible EndpointMode = "compatible"
	// EndpointModeNative Anthropic 原生模式
	EndpointModeNative EndpointMode = "native"
)

const (
	ProviderClaude       = "claude"
	DefaultClaudeBaseURL = "https://api.anthropic.com/v1"

	DefaultClaudeModel  = "claude-sonnet-4-5-20250929"
	ClaudeSonnet45Alias = "claude-sonnet-4-5"
	ClaudeHaiku45       = "claude-haiku-4-5-20251001"
	ClaudeOpus45        = "claude-opus-4-5-20251101"

	ClaudeSonnet4 = "claude-sonnet-4-20250514"
	ClaudeOpus4   = "claude-opus-4-20250514"
	ClaudeOpus41  = "claude-opus-4-1-20250805"

	AnthropicAPIVersion = "2023-06-01"
)

var claude4CachingModels = []string{
	"claude-sonnet-4-5",
	"claude-haiku-4-5",
	"claude-opus-4-5",
	"claude-opus-4-1",
	"claude-opus-4",
	"claude-sonnet-4",
}

// 各模型最小可缓存 token 数
var minCacheableTokens = map[string]int{
	"claude-opus-4-5":   4096,
	"claude-sonnet-4-5": 1024,
	"claude-haiku-4-5":  4096,
	"claude-opus-4-1":   1024,
	"claude-opus-4":     1024,
	"claude-sonnet-4":   1024,
}

type ClaudeClient struct {
	*Client
	endpointMode EndpointMode
	detectedMode EndpointMode // 实际检测到的模式
	cache        *claudeCache // 缓存状态跟踪
}

// claudeCache 跟踪缓存状态
type claudeCache struct {
	lastCacheCreation time.Time
	hasValidCache     bool
}

// NewClaudeClient creates Claude client (backward compatible)
func NewClaudeClient() AIClient {
	return NewClaudeClientWithOptions()
}

// NewClaudeClientWithOptions creates Claude client with options
func NewClaudeClientWithOptions(opts ...ClientOption) AIClient {
	// 默认选项
	claudeOpts := []ClientOption{
		WithProvider(ProviderClaude),
		WithModel(DefaultClaudeModel),
		WithBaseURL(DefaultClaudeBaseURL),
		WithEndpointMode(EndpointModeAuto), // 默认自动检测
	}

	// 合并用户选项（用户选项优先级更高）
	allOpts := append(claudeOpts, opts...)

	// 创建基础客户端
	baseClient := NewClient(allOpts...).(*Client)

	// 创建 Claude 客户端
	claudeClient := &ClaudeClient{
		Client:       baseClient,
		endpointMode: EndpointModeAuto,
		cache:        &claudeCache{},
	}

	// 检查是否有自定义 endpoint mode
	if baseClient.config.EndpointMode != "" {
		claudeClient.endpointMode = EndpointMode(baseClient.config.EndpointMode)
	}

	// 设置 hooks
	baseClient.hooks = claudeClient

	return claudeClient
}

// SetAPIKey 设置 API Key，同时进行端点检测
func (c *ClaudeClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	c.APIKey = apiKey

	if len(apiKey) > 8 {
		c.logger.Infof("🔧 [MCP] Claude API Key: %s...%s", apiKey[:4], apiKey[len(apiKey)-4:])
	}

	if customURL != "" {
		c.BaseURL = customURL
		c.logger.Infof("🔧 [MCP] Claude using custom BaseURL: %s", customURL)

		// 自动检测端点类型
		if c.endpointMode == EndpointModeAuto {
			c.detectEndpointMode()
		}
	} else {
		c.BaseURL = DefaultClaudeBaseURL
		c.detectedMode = EndpointModeNative
		c.logger.Infof("🔧 [MCP] Claude using official Anthropic endpoint (native mode)")
	}

	if customModel != "" {
		c.Model = customModel
		c.logger.Infof("🔧 [MCP] Claude using custom Model: %s", customModel)
	} else {
		c.logger.Infof("🔧 [MCP] Claude using default Model: %s", c.Model)
	}

	// 记录最终模式
	mode := c.endpointMode
	if mode == EndpointModeAuto {
		mode = c.detectedMode
	}
	c.logger.Infof("🎯 [MCP] Claude endpoint mode: %s", mode)
}

// detectEndpointMode 自动检测代理支持的 API 格式
func (c *ClaudeClient) detectEndpointMode() {
	c.logger.Infof("🔍 [MCP] Detecting endpoint mode for: %s", c.BaseURL)

	// 方法 1: 通过 URL 特征判断
	urlLower := strings.ToLower(c.BaseURL)
	if strings.Contains(urlLower, "anthropic.com") {
		c.detectedMode = EndpointModeNative
		c.logger.Infof("✅ [MCP] Detected Anthropic official endpoint (native)")
		return
	}

	// 方法 2: 发送探测请求
	detected := c.probeEndpoint()
	c.detectedMode = detected

	switch detected {
	case EndpointModeNative:
		c.logger.Infof("✅ [MCP] Detected native Anthropic endpoint (prompt caching supported)")
	case EndpointModeCompatible:
		c.logger.Infof("ℹ️  [MCP] Detected OpenAI compatible endpoint")
	default:
		// 默认使用兼容模式（最安全，第三方代理通常支持）
		c.detectedMode = EndpointModeCompatible
		c.logger.Infof("⚠️  [MCP] Could not detect endpoint type, defaulting to OpenAI compatible mode")
	}
}

// probeEndpoint 探测端点类型
func (c *ClaudeClient) probeEndpoint() EndpointMode {
	// 尝试原生端点
	if c.testNativeEndpoint() {
		return EndpointModeNative
	}

	// 尝试兼容端点
	if c.testCompatibleEndpoint() {
		return EndpointModeCompatible
	}

	return EndpointModeCompatible // 默认
}

// testNativeEndpoint 测试 Anthropic 原生端点
func (c *ClaudeClient) testNativeEndpoint() bool {
	baseURL := strings.TrimSuffix(c.BaseURL, "/")

	var testURL string
	if strings.HasSuffix(baseURL, "/messages") {
		testURL = baseURL
	} else if strings.HasSuffix(baseURL, "/v1") {
		testURL = baseURL + "/messages"
	} else {
		testURL = baseURL + "/messages"
	}

	c.logger.Debugf("🔍 [MCP] Testing native endpoint: %s", testURL)
	c.logger.Debugf("🔍 [MCP] Using model: %s", c.Model)

	testBody := map[string]any{
		"model":      c.Model,
		"max_tokens": 10,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	jsonData, _ := json.Marshal(testBody)
	c.logger.Debugf("🔍 [MCP] Request body: %s", string(jsonData))

	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Debugf("❌ [MCP] Failed to create native endpoint request: %v", err)
		return false
	}

	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", AnthropicAPIVersion)
	req.Header.Set("content-type", "application/json")

	c.logger.Debugf("🔍 [MCP] Request headers: x-api-key=*** anthropic-version=%s", AnthropicAPIVersion)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Debugf("❌ [MCP] Native endpoint request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.logger.Debugf("🔍 [MCP] Response status: %d", resp.StatusCode)
	c.logger.Debugf("🔍 [MCP] Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		c.logger.Debugf("❌ [MCP] Native endpoint returned status: %d", resp.StatusCode)
		return false
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		c.logger.Debugf("❌ [MCP] Failed to parse native endpoint response: %v", err)
		return false
	}

	_, hasContent := result["content"]
	_, hasChoices := result["choices"]

	if hasContent {
		c.logger.Debugf("✅ [MCP] Native endpoint detected with content field")
		return true
	}

	if hasChoices {
		c.logger.Debugf("⚠️  [MCP] Endpoint returned OpenAI format (choices), not native")
		return false
	}

	c.logger.Debugf("❌ [MCP] Unknown response format")
	return false
}

// testCompatibleEndpoint 测试 OpenAI 兼容端点
func (c *ClaudeClient) testCompatibleEndpoint() bool {
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
		"max_tokens":  10,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}

	jsonData, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", testURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Debugf("❌ [MCP] Failed to create compatible endpoint request: %v", err)
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("content-type", "application/json")

	c.logger.Debugf("🔍 [MCP] Using Authorization: Bearer ***")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Debugf("❌ [MCP] Compatible endpoint request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.logger.Debugf("🔍 [MCP] Compatible response status: %d", resp.StatusCode)
	c.logger.Debugf("🔍 [MCP] Compatible response body: %s", string(body))

	if resp.StatusCode == http.StatusOK {
		c.logger.Debugf("✅ [MCP] Compatible endpoint detected")
	} else {
		c.logger.Debugf("❌ [MCP] Compatible endpoint returned status: %d", resp.StatusCode)
	}

	return resp.StatusCode == http.StatusOK
}

// GetEndpointMode 获取当前端点模式
func (c *ClaudeClient) GetEndpointMode() EndpointMode {
	if c.endpointMode == EndpointModeAuto {
		return c.detectedMode
	}
	return c.endpointMode
}

// IsPromptCachingEnabled 检查是否启用了 Prompt Caching
func (c *ClaudeClient) IsPromptCachingEnabled() bool {
	return c.GetEndpointMode() == EndpointModeNative && c.supportsPromptCaching()
}

// setAuthHeader Claude 使用 x-api-key header
func (c *ClaudeClient) setAuthHeader(reqHeaders http.Header) {
	mode := c.GetEndpointMode()

	if mode == EndpointModeNative {
		// Anthropic 原生格式
		reqHeaders.Set("x-api-key", c.APIKey)
		reqHeaders.Set("anthropic-version", AnthropicAPIVersion)
	} else {
		// OpenAI 兼容格式
		reqHeaders.Set("Authorization", "Bearer "+c.APIKey)
	}
}

// buildUrl 构建请求 URL
func (c *ClaudeClient) buildUrl() string {
	mode := c.GetEndpointMode()
	baseURL := strings.TrimSuffix(c.BaseURL, "/")

	if mode == EndpointModeNative {
		if strings.HasSuffix(baseURL, "/messages") {
			return baseURL
		}
		return baseURL + "/messages"
	}

	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

// buildMCPRequestBody 根据模式构建请求体
func (c *ClaudeClient) buildMCPRequestBody(systemPrompt, userPrompt string) map[string]any {
	mode := c.GetEndpointMode()

	switch mode {
	case EndpointModeNative:
		return c.buildNativeRequest(systemPrompt, userPrompt)
	case EndpointModeCompatible:
		return c.buildCompatibleRequest(systemPrompt, userPrompt)
	default:
		return c.buildCompatibleRequest(systemPrompt, userPrompt)
	}
}

// buildNativeRequest Anthropic 原生格式 + Prompt Caching
func (c *ClaudeClient) buildNativeRequest(systemPrompt, userPrompt string) map[string]any {
	requestBody := map[string]any{
		"model":      c.Model,
		"max_tokens": c.MaxTokens,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
	}

	// 添加 temperature
	if c.config.Temperature > 0 {
		requestBody["temperature"] = c.config.Temperature
	}

	// System prompt 使用 caching
	if systemPrompt != "" {
		if c.supportsPromptCaching() && c.shouldUseCaching(systemPrompt) {
			// 使用 caching 格式
			requestBody["system"] = []map[string]any{
				{
					"type": "text",
					"text": systemPrompt,
					"cache_control": map[string]string{
						"type": "ephemeral",
					},
				},
			}
			c.logger.Debugf("🚀 [MCP] Prompt caching enabled for system prompt")
		} else {
			// 不使用 caching
			requestBody["system"] = systemPrompt
		}
	}

	return requestBody
}

// buildCompatibleRequest OpenAI 兼容格式
func (c *ClaudeClient) buildCompatibleRequest(systemPrompt, userPrompt string) map[string]any {
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

// supportsPromptCaching 检查模型是否支持 Prompt Caching
func (c *ClaudeClient) supportsPromptCaching() bool {
	modelLower := strings.ToLower(c.Model)

	for _, supported := range claude4CachingModels {
		if strings.Contains(modelLower, supported) {
			return true
		}
	}
	return false
}

// shouldUseCaching 判断是否应使用 caching
func (c *ClaudeClient) shouldUseCaching(systemPrompt string) bool {
	// 估算 token 数（粗略估计：1 token ≈ 4 字符）
	estimatedTokens := len(systemPrompt) / 4
	minTokens := c.getMinCacheableTokens()

	if estimatedTokens < minTokens {
		c.logger.Debugf("ℹ️  [MCP] System prompt too short for caching (%d < %d tokens)",
			estimatedTokens, minTokens)
		return false
	}

	return true
}

// getMinCacheableTokens 获取模型最小可缓存 token 数
func (c *ClaudeClient) getMinCacheableTokens() int {
	modelLower := strings.ToLower(c.Model)

	// 精确匹配
	if tokens, ok := minCacheableTokens[modelLower]; ok {
		return tokens
	}

	// 前缀匹配
	for model, tokens := range minCacheableTokens {
		if strings.HasPrefix(modelLower, model) {
			return tokens
		}
	}

	// 默认 1024
	return 1024
}

// parseMCPResponse 根据模式解析响应
func (c *ClaudeClient) parseMCPResponse(body []byte) (string, error) {
	mode := c.GetEndpointMode()

	switch mode {
	case EndpointModeNative:
		return c.parseNativeResponse(body)
	case EndpointModeCompatible:
		return c.parseCompatibleResponse(body)
	default:
		// 尝试自动识别
		return c.parseAutoResponse(body)
	}
}

// parseNativeResponse 解析 Anthropic 原生响应
func (c *ClaudeClient) parseNativeResponse(body []byte) (string, error) {
	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse Claude native response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("Claude API error: %s - %s", response.Error.Type, response.Error.Message)
	}

	// 记录缓存使用情况
	if response.Usage.CacheReadInputTokens > 0 {
		// 计算节省的成本（缓存命中只需 10% 价格）
		savedTokens := response.Usage.CacheReadInputTokens
		c.logger.Infof("💰 [MCP] Cache hit! %d tokens read from cache (90%% cost saved)", savedTokens)
	}
	if response.Usage.CacheCreationInputTokens > 0 {
		c.logger.Debugf("💾 [MCP] Cache created: %d tokens", response.Usage.CacheCreationInputTokens)
		c.cache.lastCacheCreation = time.Now()
		c.cache.hasValidCache = true
	}

	// 报告 token 使用
	totalTokens := response.Usage.InputTokens + response.Usage.OutputTokens
	if TokenUsageCallback != nil && totalTokens > 0 {
		TokenUsageCallback(TokenUsage{
			Provider:         c.Provider,
			Model:            c.Model,
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      totalTokens,
		})
	}

	for _, content := range response.Content {
		if content.Type == "text" {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in Claude response")
}

// parseCompatibleResponse 解析 OpenAI 兼容响应
func (c *ClaudeClient) parseCompatibleResponse(body []byte) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI compatible response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// 报告 token 使用
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

// parseAutoResponse 自动识别响应格式
func (c *ClaudeClient) parseAutoResponse(body []byte) (string, error) {
	// 尝试 Anthropic 格式
	var nativeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &nativeResp); err == nil && len(nativeResp.Content) > 0 {
		c.logger.Debugf("🔍 [MCP] Auto-detected Anthropic response format")
		c.detectedMode = EndpointModeNative
		return c.parseNativeResponse(body)
	}

	// 尝试 OpenAI 格式
	var compatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &compatResp); err == nil && len(compatResp.Choices) > 0 {
		c.logger.Debugf("🔍 [MCP] Auto-detected OpenAI compatible format")
		c.detectedMode = EndpointModeCompatible
		return c.parseCompatibleResponse(body)
	}

	return "", fmt.Errorf("unable to parse response: unknown format")
}
