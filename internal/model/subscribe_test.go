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
