package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLangTag(t *testing.T) {
	assert.Equal(t, "zh-cn", NormalizeLangTag("zh_CN"))
	assert.Equal(t, "en-us", NormalizeLangTag("EN_US"))
	assert.Equal(t, "zh-hans", NormalizeLangTag(`"zh-Hans"`))
	assert.Equal(t, "ja", NormalizeLangTag("  JA  "))
}

func TestExtractXMLLanguage(t *testing.T) {
	assert.Equal(t, "zh-cn", ExtractXMLLanguage(`<html lang="zh-CN">`))
	assert.Equal(t, "en-us", ExtractXMLLanguage(`<div xml:lang="en-US">`))
	assert.Equal(t, "ja", ExtractXMLLanguage(`<feed xml:lang='ja'>`))
	assert.Equal(t, "", ExtractXMLLanguage(`<p>plain text</p>`))
}

func TestMatchLanguage(t *testing.T) {
	// Chinese matching
	assert.True(t, MatchLanguage("zh-CN", "zh"))
	assert.True(t, MatchLanguage("zh-TW", "zh"))
	assert.True(t, MatchLanguage("zh-Hans", "zh-CN"))
	assert.True(t, MatchLanguage("zh-Hant", "zh-TW"))
	assert.True(t, MatchLanguage("zh-HK", "zh-TW"))
	assert.False(t, MatchLanguage("zh-TW", "zh-CN")) // Traditional vs Simplified
	assert.False(t, MatchLanguage("en", "zh"))

	// English matching
	assert.True(t, MatchLanguage("en-US", "en"))
	assert.True(t, MatchLanguage("en-GB", "en-US"))
	assert.True(t, MatchLanguage("en", "en-GB"))

	// Other languages
	assert.True(t, MatchLanguage("ja-JP", "ja"))
	assert.True(t, MatchLanguage("ru-RU", "ru"))
	assert.True(t, MatchLanguage("fr-FR", "fr"))
	assert.True(t, MatchLanguage("de-DE", "de"))
	assert.False(t, MatchLanguage("ja", "ko"))
	assert.False(t, MatchLanguage("fr", "de"))
}

func TestIsSameLanguage(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		targetLang string
		expected   bool
	}{
		// Chinese
		{"pure chinese", "你好世界", "zh", true},
		{"chinese blog title", "这是一篇中文博客文章测试", "zh-CN", true},
		{"chinese with tech name", "Go 1.22 正式发布，新特性解析", "zh", true},
		{"chinese with brand", "OpenAI 发布 GPT-4.5 模型，性能大幅提升", "zh", true},
		{"chinese with punctuation", "【快讯】苹果发布新一代芯片：性能提升30%！", "zh", true},
		{"english to chinese", "Hello world, this is a test.", "zh", false},
		{"japanese with kanji to chinese", "日本の首都は東京です。最新のAI技術を紹介します。", "zh", false},
		{"japanese news title to chinese", "新機能のお知らせ", "zh", false},

		// English
		{"pure english", "Hello world, this is a test.", "en", true},
		{"english with region", "This is English", "en-US", true},
		{"english with smart quotes", "Apple’s latest M4 chip is here, it’s amazing!", "en", true},
		{"english with loanword", "We visited a nice café and looked at his résumé.", "en", true},
		{"chinese to english", "你好世界", "en", false},
		{"french with accents to english", "Bonjour le monde, c'est l'été à Paris", "en", false},

		// Japanese
		{"japanese hiragana katakana", "こんにちは世界、テストです", "ja", true},
		{"japanese with kanji", "最新技術動向のレポートです", "ja", true},
		{"english to japanese", "Hello world", "ja", false},
		{"chinese to japanese", "人工智能最新技术进展报告", "ja", false},

		// Korean
		{"korean hangul", "안녕하세요 세계", "ko", true},
		{"english to korean", "Hello world", "ko", false},
		{"chinese to korean", "你好世界", "ko", false},

		// Russian / Cyrillic
		{"russian cyrillic", "Привет мир, новости технологий", "ru", true},
		{"english to russian", "Hello world", "ru", false},

		// Arabic
		{"arabic text", "مرحبا بالعالم", "ar", true},
		{"english to arabic", "Hello world", "ar", false},

		// Hebrew
		{"hebrew text", "שלום עולם", "he", true},
		{"english to hebrew", "Hello world", "he", false},

		// Greek
		{"greek text", "Γειά σου Κόσμε", "el", true},
		{"english to greek", "Hello world", "el", false},

		// Thai
		{"thai text", "สวัสดีชาวโลก", "th", true},
		{"english to thai", "Hello world", "th", false},

		// Hindi
		{"hindi devanagari", "नमस्ते दुनिया", "hi", true},
		{"english to hindi", "Hello world", "hi", false},

		// German
		{"german with umlauts", "Schöne Grüße aus München und Köln", "de", true},

		// French
		{"french with accents", "Très bien, c'est l'été à Paris", "fr", true},

		// Spanish
		{"spanish with tilde", "¡Hola! ¿Cómo estás? El niño está feliz.", "es", true},

		// Vietnamese
		{"vietnamese text", "Xin chào thế giới, đây là bài viết mới", "vi", true},

		// Empty and digits
		{"empty string", "", "zh", true},
		{"whitespace only", "   ", "en", true},
		{"digits and symbols only", "2026-08-28 10:00:00 #12345", "zh", true},
		{"numbers only", "1234567890", "en", true},

		// HTML stripped
		{"html wrapped chinese", "<p><a href='https://example.com'>你好世界</a></p>", "zh", true},
		{"html wrapped english", "<div><span>Hello world</span></div>", "en", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSameLanguage(tt.text, tt.targetLang))
		})
	}
}

func TestIsSameLanguageWithMeta(t *testing.T) {
	// XML metadata indicates same language -> skip translation
	assert.True(t, IsSameLanguageWithMeta("Some ambiguous text", "en", "en-US"))
	assert.True(t, IsSameLanguageWithMeta("Some ambiguous text", "fr", "fr-FR"))
	assert.True(t, IsSameLanguageWithMeta("Some ambiguous text", "de", "de"))
	assert.True(t, IsSameLanguageWithMeta("Tech News", "zh", "zh-CN"))

	// XML metadata indicates different language -> translate
	assert.False(t, IsSameLanguageWithMeta("Tech News", "zh", "en"))
	assert.False(t, IsSameLanguageWithMeta("Tech News", "ja", "en"))

	// Embedded XML lang attribute inside text
	assert.True(t, IsSameLanguageWithMeta(`<article lang="zh-CN">Hello Tech</article>`, "zh", ""))
	assert.True(t, IsSameLanguageWithMeta(`<div xml:lang="ja">Title</div>`, "ja", ""))
}
