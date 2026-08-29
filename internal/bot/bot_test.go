package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/translate"
)

type mockTranslator struct {
	shouldFail       bool
	callCount        int
	contentCallCount int
}

func (m *mockTranslator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	m.callCount++
	if m.shouldFail {
		return "", errors.New("mock translate error")
	}
	return "translated:" + text, nil
}

func (m *mockTranslator) TranslateContent(ctx context.Context, title, preview, targetLang string) (string, string, error) {
	m.contentCallCount++
	if m.shouldFail {
		return "", "", errors.New("mock translate content error")
	}
	return "translated:" + title, "translated:" + preview, nil
}

func TestTranslateContent_SuccessCaching(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	title, preview := b.translateContent("hash123", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "translated:Original Title", title)
	assert.Equal(t, "translated:Original Preview", preview)
	assert.Equal(t, 1, mockTr.contentCallCount) // combined in 1 call

	// Second call with identical content should hit full cache and make 0 LLM calls
	title2, preview2 := b.translateContent("hash123", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "translated:Original Title", title2)
	assert.Equal(t, "translated:Original Preview", preview2)
	assert.Equal(t, 1, mockTr.contentCallCount) // still 1
	assert.Equal(t, 0, mockTr.callCount)
}

func TestTranslateContent_PartialUpdate(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// First call
	b.translateContent("hash123", "zh", "Original Title", "Original Preview")
	assert.Equal(t, 1, mockTr.contentCallCount)

	// Second call with title changed but preview unchanged -> only translates title (1 Translate call)
	title, preview := b.translateContent("hash123", "zh", "New Title", "Original Preview")
	assert.Equal(t, "translated:New Title", title)
	assert.Equal(t, "translated:Original Preview", preview) // reused from cache!
	assert.Equal(t, 1, mockTr.contentCallCount)
	assert.Equal(t, 1, mockTr.callCount)
}

func TestTranslateContent_AlreadyTargetLanguageSkipped(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// Chinese content with Chinese target language -> 0 calls to translator!
	title, preview := b.translateContent("hash_zh", "zh", "你好世界，这是一篇中文测试文章", "这里是文章的正文预览内容")
	assert.Equal(t, "你好世界，这是一篇中文测试文章", title)
	assert.Equal(t, "这里是文章的正文预览内容", preview)
	assert.Equal(t, 0, mockTr.contentCallCount)
	assert.Equal(t, 0, mockTr.callCount)

	// Subsequent call should also hit cache with 0 translator calls
	title2, preview2 := b.translateContent("hash_zh", "zh", "你好世界，这是一篇中文测试文章", "这里是文章的正文预览内容")
	assert.Equal(t, "你好世界，这是一篇中文测试文章", title2)
	assert.Equal(t, "这里是文章的正文预览内容", preview2)
	assert.Equal(t, 0, mockTr.contentCallCount)
	assert.Equal(t, 0, mockTr.callCount)
}

func TestTranslateContent_PartialTargetLanguage(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// Title is English, Preview is already Chinese -> Only translates Title!
	title, preview := b.translateContent("hash_mixed", "zh", "English Article Title", "这里是已经为中文的正文内容预览")
	assert.Equal(t, "translated:English Article Title", title)
	assert.Equal(t, "这里是已经为中文的正文内容预览", preview)
	assert.Equal(t, 0, mockTr.contentCallCount) // Did not need combined translation
	assert.Equal(t, 1, mockTr.callCount)        // Only 1 single translation for title
}

func TestTranslateContent_XMLMetaLanguageMatch(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// Feed metadata indicates content is already in target language -> 0 calls
	title, preview := b.translateContent("hash_meta", "en", "Ambiguous Title", "Ambiguous preview content", "en-US")
	assert.Equal(t, "Ambiguous Title", title)
	assert.Equal(t, "Ambiguous preview content", preview)
	assert.Equal(t, 0, mockTr.contentCallCount)
	assert.Equal(t, 0, mockTr.callCount)
}

func TestTranslateContent_ErrorNoCaching(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: true}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	title, preview := b.translateContent("hash456", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "Original Title", title)     // fallback to original
	assert.Equal(t, "Original Preview", preview) // fallback to original
	assert.Equal(t, 1, mockTr.contentCallCount)

	// Next call should NOT hit cache because translation failed and wasn't cached
	title2, preview2 := b.translateContent("hash456", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "Original Title", title2)
	assert.Equal(t, "Original Preview", preview2)
	assert.Equal(t, 2, mockTr.contentCallCount)
}

func TestRenderContentMessage(t *testing.T) {
	b := &Bot{
		transCache: translate.NewCache(100),
	}

	source := &model.Source{
		Title: "Source Title",
	}
	sub := &model.Subscribe{
		UserID: 12345,
		Tag:    "#news",
	}
	content := &model.Content{
		HashID:      "hash_render",
		Title:       "Sample Post",
		RawLink:     "https://example.com/sample",
		Description: "<p>Hello World</p>",
	}

	msg, err := b.renderContentMessage(source, sub, content)
	assert.NoError(t, err)
	assert.Contains(t, msg, "Source Title")
	assert.Contains(t, msg, "Sample Post")
	assert.Contains(t, msg, "https://example.com/sample")
	assert.Contains(t, msg, "Hello World")
	assert.Contains(t, msg, "#news")
}

func TestRenderContentMessage_PreviewLimitAndDirection(t *testing.T) {
	b := &Bot{
		transCache: translate.NewCache(100),
	}
	source := &model.Source{Title: "Source"}
	content := &model.Content{
		HashID:      "hash_dir",
		Title:       "Post",
		RawLink:     "https://example.com",
		Description: "<p>0123456789</p>",
	}

	ptrVal := func(v int) *int { return &v }

	t.Run("positive limit takes head", func(t *testing.T) {
		sub := &model.Subscribe{PreviewLength: ptrVal(5)}
		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.Contains(t, msg, "0123…")
		assert.NotContains(t, msg, "…6789")
	})

	t.Run("negative limit takes tail", func(t *testing.T) {
		sub := &model.Subscribe{PreviewLength: ptrVal(-5)}
		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.Contains(t, msg, "…6789")
		assert.NotContains(t, msg, "0123…")
	})

	t.Run("zero limit disables preview", func(t *testing.T) {
		sub := &model.Subscribe{PreviewLength: ptrVal(0)}
		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.NotContains(t, msg, "0123456789")
	})
}

type capturingTranslator struct {
	lastTitle   string
	lastPreview string
}

func (c *capturingTranslator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	return "translated:" + text, nil
}

func (c *capturingTranslator) TranslateContent(ctx context.Context, title, preview, targetLang string) (string, string, error) {
	c.lastTitle = title
	c.lastPreview = preview
	return "translated:" + title, "translated:" + preview, nil
}

func TestRenderContentMessage_TruncatesBeforeTranslation(t *testing.T) {
	mockTr := &capturingTranslator{}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	source := &model.Source{Title: "News"}
	ptrVal := func(v int) *int { return &v }

	t.Run("positive limit truncates before translation to save tokens", func(t *testing.T) {
		sub := &model.Subscribe{
			TranslateLang: "zh",
			PreviewLength: ptrVal(10),
		}
		longDesc := "This is a very long English description that would waste a lot of tokens if translated in full."
		content := &model.Content{
			HashID:      "hash_token_pos",
			Title:       "English Title",
			RawLink:     "https://example.com",
			Description: longDesc,
		}

		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.Contains(t, msg, "translated:This is a…")
		assert.Equal(t, "This is a…", mockTr.lastPreview)
		// Ensure full long description was NOT sent to translator
		assert.NotEqual(t, longDesc, mockTr.lastPreview)
	})

	t.Run("negative limit truncates before translation to save tokens", func(t *testing.T) {
		sub := &model.Subscribe{
			TranslateLang: "zh",
			PreviewLength: ptrVal(-15),
		}
		longDesc := "This is a very long English description that would waste a lot of tokens if translated in full."
		content := &model.Content{
			HashID:      "hash_token_neg",
			Title:       "English Title",
			RawLink:     "https://example.com",
			Description: longDesc,
		}

		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.Contains(t, msg, "translated:…lated in full.")
		assert.Equal(t, "…lated in full.", mockTr.lastPreview)
		assert.NotEqual(t, longDesc, mockTr.lastPreview)
	})

	t.Run("zero limit skips preview translation completely", func(t *testing.T) {
		mockTr.lastPreview = "should_not_be_called"
		sub := &model.Subscribe{
			TranslateLang: "zh",
			PreviewLength: ptrVal(0),
		}
		content := &model.Content{
			HashID:      "hash_token_zero",
			Title:       "English Title",
			RawLink:     "https://example.com",
			Description: "Long description here",
		}

		msg, err := b.renderContentMessage(source, sub, content)
		assert.NoError(t, err)
		assert.Contains(t, msg, "translated:English Title")
		// Preview translation was skipped
		assert.Equal(t, "should_not_be_called", mockTr.lastPreview)
	})
}

func TestBroadcastEdit_NoDuplicateTranslationWhenContentUnchanged(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// Initial translation
	t1, p1 := b.translateContent("hash_edit", "zh", "Title 1", "Preview 1")
	assert.Equal(t, "translated:Title 1", t1)
	assert.Equal(t, "translated:Preview 1", p1)
	assert.Equal(t, 1, mockTr.contentCallCount)

	// Simulate polling/broadcast edit with unchanged title and preview
	t1Again, p1Again := b.translateContent("hash_edit", "zh", "Title 1", "Preview 1")
	assert.Equal(t, "translated:Title 1", t1Again)
	assert.Equal(t, "translated:Preview 1", p1Again)
	// Zero new calls to translator!
	assert.Equal(t, 1, mockTr.contentCallCount)
	assert.Equal(t, 0, mockTr.callCount)
}
