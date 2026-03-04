package mcp

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func stringToReadCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func TestClaudeClient_CachingFallback(t *testing.T) {
	endpointCachingStatusMu.Lock()
	endpointCachingStatus = make(map[string]bool)
	endpointCachingStatusMu.Unlock()

	t.Run("caching error triggers fallback", func(t *testing.T) {
		mockHTTP := NewMockHTTPClient()
		mockLogger := NewMockLogger()

		callCount := 0
		mockHTTP.ResponseFunc = func(req *http.Request) (*http.Response, error) {
			callCount++

			bodyBytes := make([]byte, 65536)
			n, _ := req.Body.Read(bodyBytes)
			bodyStr := string(bodyBytes[:n])

			if callCount == 1 {
				if !strings.Contains(bodyStr, "system") || strings.Contains(bodyStr, "role\":\"system\"") {
					t.Errorf("First request should use native system field (not messages array), got: %s", bodyStr)
				}
				return &http.Response{
					StatusCode: 500,
					Body:       stringToReadCloser(`{"error":{"type":"invalid_request","message":"cache_control not supported"}}`),
					Header:     make(http.Header),
				}, nil
			}

			if callCount == 2 {
				if strings.Contains(bodyStr, "cache_control") {
					t.Error("Second request should NOT use caching format")
				}
				return &http.Response{
					StatusCode: 200,
					Body:       stringToReadCloser(`{"content":[{"type":"text","text":"success"}]}`),
					Header:     make(http.Header),
				}, nil
			}

			return nil, nil
		}

		client := NewClaudeClientWithOptions(
			WithHTTPClient(mockHTTP.ToHTTPClient()),
			WithLogger(mockLogger),
			WithAPIKey("test-key"),
			WithBaseURL("https://third-party-proxy.com/v1"),
			WithModel("claude-opus-4-6"),
			WithEndpointMode(EndpointModeNative),
		).(*ClaudeClient)

		longSystemPrompt := strings.Repeat("a", 5000)

		_, err := client.callWithCachingFallback(longSystemPrompt, "test prompt")

		if callCount != 2 {
			t.Errorf("Expected 2 calls (first with caching, second without), got %d", callCount)
		}

		endpointCachingStatusMu.RLock()
		supports, known := endpointCachingStatus["https://third-party-proxy.com/v1"]
		endpointCachingStatusMu.RUnlock()
		if !known || supports {
			t.Error("Global state should mark endpoint as not supporting caching")
		}

		if err != nil {
			t.Errorf("Second call should succeed, got error: %v", err)
		}
	})

	t.Run("non-caching error does not trigger fallback", func(t *testing.T) {
		mockHTTP := NewMockHTTPClient()
		mockHTTP.SetErrorResponse(401, "Unauthorized")

		client := NewClaudeClientWithOptions(
			WithHTTPClient(mockHTTP.ToHTTPClient()),
			WithAPIKey("test-key"),
			WithBaseURL("https://api.anthropic.com/v1"),
		).(*ClaudeClient)

		_, err := client.callWithCachingFallback("system", "test")

		if err == nil {
			t.Error("Should return error")
		}
	})

	t.Run("isCachingError detects caching keywords", func(t *testing.T) {
		client := &ClaudeClient{}

		testCases := []struct {
			errStr    string
			isCaching bool
		}{
			{"cache_control not supported", true},
			{"prompt caching unavailable", true},
			{"improperly formed request", true},
			{"anthropic-beta header required", true},
			{"unauthorized", false},
			{"rate limit exceeded", false},
		}

		for _, tc := range testCases {
			err := &testError{msg: tc.errStr}
			result := client.isCachingError(err)
			if result != tc.isCaching {
				t.Errorf("isCachingError(%q) = %v, want %v", tc.errStr, result, tc.isCaching)
			}
		}
	})

	t.Run("global state shared across clients", func(t *testing.T) {
		endpointCachingStatusMu.Lock()
		endpointCachingStatus = make(map[string]bool)
		endpointCachingStatusMu.Unlock()

		mockHTTP := NewMockHTTPClient()
		callCount := 0
		mockHTTP.ResponseFunc = func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: 200,
				Body:       stringToReadCloser(`{"content":[{"type":"text","text":"success"}]}`),
				Header:     make(http.Header),
			}, nil
		}

		_ = NewClaudeClientWithOptions(
			WithHTTPClient(mockHTTP.ToHTTPClient()),
			WithAPIKey("test-key"),
			WithBaseURL("https://shared-proxy.com/v1"),
			WithModel("claude-opus-4-6"),
			WithEndpointMode(EndpointModeNative),
		)

		longSystemPrompt := strings.Repeat("a", 5000)

		endpointCachingStatusMu.Lock()
		endpointCachingStatus["https://shared-proxy.com/v1"] = false
		endpointCachingStatusMu.Unlock()

		client2 := NewClaudeClientWithOptions(
			WithHTTPClient(mockHTTP.ToHTTPClient()),
			WithAPIKey("test-key-2"),
			WithBaseURL("https://shared-proxy.com/v1"),
			WithModel("claude-opus-4-6"),
			WithEndpointMode(EndpointModeNative),
		).(*ClaudeClient)

		_, err := client2.callWithCachingFallback(longSystemPrompt, "test prompt")
		if err != nil {
			t.Errorf("Second client should succeed, got error: %v", err)
		}

		if callCount != 1 {
			t.Errorf("Second client should make only 1 request (no caching retry), got %d", callCount)
		}

		bodyBytes := make([]byte, 65536)
		n, _ := mockHTTP.Requests[0].Body.Read(bodyBytes)
		bodyStr := string(bodyBytes[:n])
		if strings.Contains(bodyStr, "cache_control") {
			t.Error("Second client should NOT use caching format due to global state")
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
