package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/model"
)

func TestSortContentUpdatesOldestFirst(t *testing.T) {
	oldest := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	newest := oldest.Add(48 * time.Hour)
	updates := []contentUpdate{
		{content: &model.Content{Title: "newest", PublishedAt: &newest}},
		{content: &model.Content{Title: "undated"}},
		{content: &model.Content{Title: "oldest", PublishedAt: &oldest}},
	}

	sortContentUpdatesOldestFirst(updates)
	assert.Equal(t, []string{"undated", "oldest", "newest"}, []string{
		updates[0].content.Title, updates[1].content.Title, updates[2].content.Title,
	})
}
