package scheduler

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"

	"github.com/indes/flowerss-bot/internal/feed"
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

func TestSaveNewContentsWithoutGUID(t *testing.T) {
	source := &model.Source{
		ID:   1,
		Link: "https://example.com/feed.xml",
	}

	hashID1 := model.GenHashID(source.Link, feed.ItemGUID(&gofeed.Item{
		Title: "Item 1",
		Link:  "https://example.com/item1",
	}))
	hashID2 := model.GenHashID(source.Link, feed.ItemGUID(&gofeed.Item{
		Title: "Item 2",
		Link:  "https://example.com/item2",
	}))

	assert.NotEqual(t, hashID1, hashID2, "Hash IDs for items without GUID should be distinct based on link fallback")
}

