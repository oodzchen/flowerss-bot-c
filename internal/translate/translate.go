// Package translate provides pluggable text translation used before RSS
// contents are pushed to subscribers. The default implementation talks to any
// OpenAI-compatible chat completions API (DeepSeek, OpenAI, Ollama, GLM, ...).
package translate

import (
	"context"
	"strings"
	"sync"

	"github.com/indes/flowerss-bot/internal/config"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/pkg/client"
)

// Translator translates a piece of text into the target language.
type Translator interface {
	// Translate translates text into targetLang (a language code such as
	// "zh", "en", "ja"). An empty or already-target text is returned as-is.
	Translate(ctx context.Context, text, targetLang string) (string, error)
}

// NewFromConfig builds the Translator configured in config.yml. It returns nil
// when the configured provider is unknown, in which case the bot simply skips
// translation.
func NewFromConfig(httpClient *client.HttpClient) Translator {
	switch strings.ToLower(config.TranslateProvider) {
	case "llm", "openai", "deepseek", "openrouter", "":
		t := NewLLMTranslator(
			httpClient,
			config.TranslateBaseURL,
			config.TranslateAPIKey,
			config.TranslateModel,
		)
		if config.TranslateHTTPReferer != "" {
			t.SetHTTPReferer(config.TranslateHTTPReferer)
		}
		if config.TranslateXTitle != "" {
			t.SetXTitle(config.TranslateXTitle)
		}
		log.Infof(
			"init translate: provider=%s model=%s base_url=%s api_key=%t",
			config.TranslateProvider, t.model, t.baseURL, t.apiKey != "",
		)
		return t
	default:
		log.Warnf("unknown translate provider %q, translation disabled", config.TranslateProvider)
		return nil
	}
}

// NormalizeLang normalizes a user-supplied language argument. Disable words
// map to an empty string, everything else is lowercased and trimmed.
func NormalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "", "off", "none", "close", "关闭", "不翻译", "no":
		return ""
	}
	return lang
}

// langNames maps common language codes to natural language names. LLMs
// understand names better than ISO codes, so the target language is sent as a
// name whenever possible.
var langNames = map[string]string{
	"zh":      "Chinese (Simplified)",
	"zh-cn":   "Chinese (Simplified)",
	"zh-hans": "Chinese (Simplified)",
	"zh-sg":   "Chinese (Simplified)",
	"zh-tw":   "Chinese (Traditional)",
	"zh-hant": "Chinese (Traditional)",
	"zh-hk":   "Chinese (Traditional)",
	"en":      "English",
	"en-us":   "English",
	"en-gb":   "English",
	"ja":      "Japanese",
	"ko":      "Korean",
	"fr":      "French",
	"de":      "German",
	"ru":      "Russian",
	"es":      "Spanish",
	"pt":      "Portuguese",
	"it":      "Italian",
	"nl":      "Dutch",
	"ar":      "Arabic",
	"hi":      "Hindi",
	"tr":      "Turkish",
	"th":      "Thai",
	"vi":      "Vietnamese",
	"id":      "Indonesian",
	"pl":      "Polish",
	"uk":      "Ukrainian",
	"sv":      "Swedish",
	"da":      "Danish",
	"fi":      "Finnish",
	"no":      "Norwegian",
	"cs":      "Czech",
	"el":      "Greek",
	"he":      "Hebrew",
	"ro":      "Romanian",
	"hu":      "Hungarian",
	"bg":      "Bulgarian",
	"ms":      "Malay",
	"fa":      "Persian",
}

// LanguageName returns a natural language name for a language code, falling
// back to the raw code when it is unknown.
func LanguageName(lang string) string {
	if name, ok := langNames[strings.ToLower(lang)]; ok {
		return name
	}
	return lang
}

// CachedTranslation holds the translated title and preview of one feed item in
// one target language, so content pushed to many subscribers only triggers one
// translation per (item, language) pair.
type CachedTranslation struct {
	Title   string
	Preview string
}

// Cache is a small in-memory translation cache. It is safe for concurrent use
// and evicts everything once it grows past maxSize (simple, bounded memory).
type Cache struct {
	mu      sync.Mutex
	items   map[string]CachedTranslation
	maxSize int
}

func NewCache(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 2000
	}
	return &Cache{items: make(map[string]CachedTranslation), maxSize: maxSize}
}

func (c *Cache) Get(key string) (CachedTranslation, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *Cache) Put(key string, v CachedTranslation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.maxSize {
		c.items = make(map[string]CachedTranslation)
	}
	c.items[key] = v
}
