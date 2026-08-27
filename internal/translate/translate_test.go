package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indes/flowerss-bot/internal/config"
	"github.com/indes/flowerss-bot/pkg/client"
)

func TestNormalizeLang(t *testing.T) {
	cases := map[string]string{
		"zh":    "zh",
		"  EN ": "en",
		"Zh-TW": "zh-tw",
		"off":   "",
		"none":  "",
		"关闭":    "",
		"":      "",
	}
	for in, want := range cases {
		assert.Equal(t, want, NormalizeLang(in), "NormalizeLang(%q)", in)
	}
}

func TestLanguageName(t *testing.T) {
	assert.Equal(t, "Chinese (Simplified)", LanguageName("zh"))
	assert.Equal(t, "Japanese", LanguageName("JA"))
	assert.Equal(t, "English", LanguageName("en-US"))
	assert.Equal(t, "xx-unknown", LanguageName("xx-unknown"))
}

func TestMaxTokensFor(t *testing.T) {
	assert.Equal(t, 512, maxTokensFor(""))
	assert.Equal(t, 4096, maxTokensFor(strings.Repeat("长", 3000)))
}

func TestLLMTranslator_Translate(t *testing.T) {
	var gotBody chatRequest
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "你好，世界"}},
			},
		})
	}))
	defer srv.Close()

	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, srv.URL, "sk-test", "test-model")
	out, err := tr.Translate(context.Background(), "Hello world", "zh")
	require.NoError(t, err)
	assert.Equal(t, "你好，世界", out)
	assert.Equal(t, "Bearer sk-test", gotAuth)
	assert.Equal(t, "test-model", gotBody.Model)
	assert.Len(t, gotBody.Messages, 2)
	assert.Equal(t, "system", gotBody.Messages[0].Role)
	assert.Contains(t, gotBody.Messages[1].Content, "Chinese (Simplified)")
	assert.Contains(t, gotBody.Messages[1].Content, "Hello world")
}

func TestLLMTranslator_TranslateContent(t *testing.T) {
	var gotBody chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "<TITLE>\n翻译标题\n</TITLE>\n<PREVIEW>\n翻译预览内容\n</PREVIEW>"}},
			},
		})
	}))
	defer srv.Close()

	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, srv.URL, "sk-test", "test-model")
	title, preview, err := tr.TranslateContent(context.Background(), "English Title", "English Preview Content", "zh")
	require.NoError(t, err)
	assert.Equal(t, "翻译标题", title)
	assert.Equal(t, "翻译预览内容", preview)
}

func TestLLMTranslator_TranslateEmpty(t *testing.T) {
	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, "http://127.0.0.1:1", "", "m")
	out, err := tr.Translate(context.Background(), "   ", "zh")
	require.NoError(t, err)
	assert.Equal(t, "   ", out)
}

func TestLLMTranslator_TranslateSameLangSkipped(t *testing.T) {
	// If text is already Chinese and target is zh, it should skip HTTP call completely
	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, "http://invalid-address-that-would-fail", "", "m")
	out, err := tr.Translate(context.Background(), "这是一篇中文博客", "zh")
	require.NoError(t, err)
	assert.Equal(t, "这是一篇中文博客", out)
}

func TestLLMTranslator_TranslateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, srv.URL, "bad", "m")
	_, err := tr.Translate(context.Background(), "hello", "zh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "chat/completions")
}

func TestLLMTranslator_TranslateNonJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<html>404 not found</html>`))
	}))
	defer srv.Close()

	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, srv.URL, "k", "m")
	_, err := tr.Translate(context.Background(), "hello", "zh")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
	assert.Contains(t, err.Error(), "404 not found")
}

func TestLLMTranslator_TranslateOpenRouter(t *testing.T) {
	var gotBody chatRequest
	var gotAuth, gotReferer, gotXTitle string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotReferer = r.Header.Get("HTTP-Referer")
		gotXTitle = r.Header.Get("X-Title")
		require.Equal(t, "/api/v1/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "gen-abc",
			"model":   gotBody.Model,
			"choices": []map[string]any{{"message": map[string]any{"content": "你好，世界"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
	defer srv.Close()

	httpClient := client.NewHttpClient()
	tr := NewLLMTranslator(httpClient, srv.URL+"/api/v1", "sk-or-v1-test", "anthropic/claude-3.5-sonnet")
	tr.SetHTTPReferer("https://example.com")
	tr.SetXTitle("flowerss-bot")

	out, err := tr.Translate(context.Background(), "Hello world", "zh")
	require.NoError(t, err)
	assert.Equal(t, "你好，世界", out)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", gotBody.Model)
	assert.Equal(t, "Bearer sk-or-v1-test", gotAuth)
	assert.Equal(t, "https://example.com", gotReferer)
	assert.Equal(t, "flowerss-bot", gotXTitle)
}

func TestNewFromConfig_OpenRouterProvider(t *testing.T) {
	httpClient := client.NewHttpClient()

	oldProvider, oldBaseURL, oldAPIKey, oldModel :=
		config.TranslateProvider, config.TranslateBaseURL, config.TranslateAPIKey, config.TranslateModel
	defer func() {
		config.TranslateProvider, config.TranslateBaseURL = oldProvider, oldBaseURL
		config.TranslateAPIKey, config.TranslateModel = oldAPIKey, oldModel
	}()

	config.TranslateProvider = "openrouter"
	config.TranslateBaseURL = "https://openrouter.ai/api/v1"
	config.TranslateAPIKey = "sk-or-v1-test"
	config.TranslateModel = "deepseek/deepseek-chat"

	tr := NewFromConfig(httpClient)
	require.NotNil(t, tr)
	llm, ok := tr.(*LLMTranslator)
	require.True(t, ok)
	assert.Equal(t, "deepseek/deepseek-chat", llm.model)
	assert.Equal(t, "https://openrouter.ai/api/v1", llm.baseURL)
}

func TestCache_LRU(t *testing.T) {
	c := NewCache(2)
	_, ok := c.Get("a")
	assert.False(t, ok)

	c.Put("a", CachedTranslation{Title: "t1", Preview: "p1"})
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "t1", v.Title)

	// Add 'b' -> cache has [b, a] (a was accessed last)
	c.Put("b", CachedTranslation{Title: "t2"})
	// Access 'a' again -> cache has [a, b]
	_, ok = c.Get("a")
	assert.True(t, ok)

	// Add 'c' -> capacity 2 exceeded, oldest 'b' should be evicted
	c.Put("c", CachedTranslation{Title: "t3"})

	// 'a' and 'c' should remain, 'b' evicted
	_, ok = c.Get("a")
	assert.True(t, ok)
	_, ok = c.Get("b")
	assert.False(t, ok)
	_, ok = c.Get("c")
	assert.True(t, ok)
}

func TestCache_DeleteByHash(t *testing.T) {
	c := NewCache(10)
	c.Put("hash1|zh", CachedTranslation{Title: "t1_zh", Preview: "p1_zh"})
	c.Put("hash1|en", CachedTranslation{Title: "t1_en", Preview: "p1_en"})
	c.Put("hash2|zh", CachedTranslation{Title: "t2_zh", Preview: "p2_zh"})

	c.DeleteByHash("hash1")

	_, ok1 := c.Get("hash1|zh")
	assert.False(t, ok1)
	_, ok2 := c.Get("hash1|en")
	assert.False(t, ok2)

	v, ok3 := c.Get("hash2|zh")
	assert.True(t, ok3)
	assert.Equal(t, "t2_zh", v.Title)
}
