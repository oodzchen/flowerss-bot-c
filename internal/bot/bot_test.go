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
	shouldFail bool
	callCount  int
}

func (m *mockTranslator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	m.callCount++
	if m.shouldFail {
		return "", errors.New("mock translate error")
	}
	return "translated:" + text, nil
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
	assert.Equal(t, 2, mockTr.callCount) // title + preview

	// Second call should hit cache and not call translator again
	title2, preview2 := b.translateContent("hash123", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "translated:Original Title", title2)
	assert.Equal(t, "translated:Original Preview", preview2)
	assert.Equal(t, 2, mockTr.callCount) // still 2
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
	assert.Equal(t, 2, mockTr.callCount)

	// Next call should NOT hit cache because translation failed and wasn't cached
	title2, preview2 := b.translateContent("hash456", "zh", "Original Title", "Original Preview")
	assert.Equal(t, "Original Title", title2)
	assert.Equal(t, "Original Preview", preview2)
	assert.Equal(t, 4, mockTr.callCount) // called again (2 + 2)
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

func TestBroadcastEdit_CacheEviction(t *testing.T) {
	mockTr := &mockTranslator{shouldFail: false}
	b := &Bot{
		translator: mockTr,
		transCache: translate.NewCache(100),
	}

	// First translation
	t1, _ := b.translateContent("hash_edit", "zh", "Title 1", "Preview 1")
	assert.Equal(t, "translated:Title 1", t1)
	assert.Equal(t, 2, mockTr.callCount)

	// Simulate cache hit before edit
	t1Cached, _ := b.translateContent("hash_edit", "zh", "Title 1", "Preview 1")
	assert.Equal(t, "translated:Title 1", t1Cached)
	assert.Equal(t, 2, mockTr.callCount)

	// Clear cache for edited hash
	b.transCache.DeleteByHash("hash_edit")

	// Translate new edited content
	t2, _ := b.translateContent("hash_edit", "zh", "Title 2 (Edited)", "Preview 2")
	assert.Equal(t, "translated:Title 2 (Edited)", t2)
	assert.Equal(t, 4, mockTr.callCount)
}

