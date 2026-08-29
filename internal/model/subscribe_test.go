package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeDisplayTitle(t *testing.T) {
	t.Run("uses source title by default", func(t *testing.T) {
		subscription := &Subscribe{}
		assert.Equal(t, "source title", subscription.DisplayTitle("source title"))
	})

	t.Run("uses trimmed subscription title", func(t *testing.T) {
		subscription := &Subscribe{Title: "  custom title  "}
		assert.Equal(t, "custom title", subscription.DisplayTitle("source title"))
	})
}

func TestSubscribe_PreviewLimit(t *testing.T) {
	ptr := func(i int) *int { return &i }

	t.Run("default when nil", func(t *testing.T) {
		sub := &Subscribe{}
		assert.Equal(t, 300, sub.GetPreviewLimit(300))
		assert.Equal(t, "默认 (前300字符)", sub.PreviewLengthDisplay(300))
		assert.Equal(t, "默认 (后300字符)", sub.PreviewLengthDisplay(-300))
		assert.Equal(t, "默认 (关闭)", sub.PreviewLengthDisplay(0))
	})

	t.Run("explicit positive limit", func(t *testing.T) {
		sub := &Subscribe{PreviewLength: ptr(400)}
		assert.Equal(t, 400, sub.GetPreviewLimit(300))
		assert.Equal(t, "前400字符", sub.PreviewLengthDisplay(300))
	})

	t.Run("explicit negative limit", func(t *testing.T) {
		sub := &Subscribe{PreviewLength: ptr(-400)}
		assert.Equal(t, -400, sub.GetPreviewLimit(300))
		assert.Equal(t, "后400字符", sub.PreviewLengthDisplay(300))
	})

	t.Run("explicit disabled limit", func(t *testing.T) {
		sub := &Subscribe{PreviewLength: ptr(0)}
		assert.Equal(t, 0, sub.GetPreviewLimit(300))
		assert.Equal(t, "关闭", sub.PreviewLengthDisplay(300))
	})
}
