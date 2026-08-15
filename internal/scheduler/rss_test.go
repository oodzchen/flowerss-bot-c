package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/feed"
	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/storage"
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

func TestIsContentEdited(t *testing.T) {
	pubTime := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	oldContent := &model.Content{
		Title:       "Original Title",
		Description: "Original Description",
		RawLink:     "https://example.com/post/1",
		PublishedAt: &pubTime,
	}

	// Identical item
	itemSame := &gofeed.Item{
		Title:           "Original Title",
		Description:     "Original Description",
		Link:            "https://example.com/post/1",
		PublishedParsed: &pubTime,
	}
	assert.False(t, isContentEdited(oldContent, itemSame))

	// Title edited
	itemTitleEdited := &gofeed.Item{
		Title:           "Updated Title",
		Description:     "Original Description",
		Link:            "https://example.com/post/1",
		PublishedParsed: &pubTime,
	}
	assert.True(t, isContentEdited(oldContent, itemTitleEdited))

	// Description edited
	itemDescEdited := &gofeed.Item{
		Title:           "Original Title",
		Description:     "Updated Description",
		Link:            "https://example.com/post/1",
		PublishedParsed: &pubTime,
	}
	assert.True(t, isContentEdited(oldContent, itemDescEdited))

	// Link edited
	itemLinkEdited := &gofeed.Item{
		Title:           "Original Title",
		Description:     "Original Description",
		Link:            "https://example.com/post/1-updated",
		PublishedParsed: &pubTime,
	}
	assert.True(t, isContentEdited(oldContent, itemLinkEdited))

	// Published time edited
	newPubTime := pubTime.Add(1 * time.Hour)
	itemTimeEdited := &gofeed.Item{
		Title:           "Original Title",
		Description:     "Original Description",
		Link:            "https://example.com/post/1",
		PublishedParsed: &newPubTime,
	}
	assert.True(t, isContentEdited(oldContent, itemTimeEdited))

	// Old content had empty description (DB upgrade backfill case) -> should not trigger edit if other fields match
	oldEmptyDesc := &model.Content{
		Title:       "Original Title",
		Description: "",
		RawLink:     "https://example.com/post/1",
		PublishedAt: &pubTime,
	}
	assert.False(t, isContentEdited(oldEmptyDesc, itemSame))
}

func TestProcessFeedItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	assert.NoError(t, err)

	userStorage := storage.NewUserStorageImpl(db)
	contentStorage := storage.NewContentStorageImpl(db)
	sourceStorage := storage.NewSourceStorageImpl(db)
	subscriptionStorage := storage.NewSubscriptionStorageImpl(db)

	ctx := context.Background()
	assert.NoError(t, userStorage.Init(ctx))
	assert.NoError(t, contentStorage.Init(ctx))
	assert.NoError(t, sourceStorage.Init(ctx))
	assert.NoError(t, subscriptionStorage.Init(ctx))

	appCore := core.NewCore(userStorage, contentStorage, sourceStorage, subscriptionStorage, nil, nil)
	task := NewRssTask(appCore)

	source := &model.Source{
		ID:   1,
		Link: "https://example.com/feed.xml",
	}

	pubTime := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	item1 := &gofeed.Item{
		GUID:            "item-1",
		Title:           "Item 1 Initial",
		Description:     "Description 1 Initial",
		Link:            "https://example.com/1",
		PublishedParsed: &pubTime,
	}

	// 1. Initial crawl: item1 is new
	newContents, editContents, err := task.processFeedItems(source, []*gofeed.Item{item1})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(newContents))
	assert.Equal(t, 0, len(editContents))
	assert.Equal(t, "Item 1 Initial", newContents[0].Title)

	// 2. Second crawl with unchanged item: 0 new, 0 edited
	newContents, editContents, err = task.processFeedItems(source, []*gofeed.Item{item1})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(newContents))
	assert.Equal(t, 0, len(editContents))

	// 3. Third crawl: item1 is edited and item2 is new
	item1Edited := &gofeed.Item{
		GUID:            "item-1",
		Title:           "Item 1 Edited Title",
		Description:     "Description 1 Edited Content",
		Link:            "https://example.com/1",
		PublishedParsed: &pubTime,
	}
	item2 := &gofeed.Item{
		GUID:            "item-2",
		Title:           "Item 2 New",
		Description:     "Description 2",
		Link:            "https://example.com/2",
		PublishedParsed: &pubTime,
	}

	newContents, editContents, err = task.processFeedItems(source, []*gofeed.Item{item1Edited, item2})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(newContents))
	assert.Equal(t, "Item 2 New", newContents[0].Title)
	assert.Equal(t, 1, len(editContents))
	assert.Equal(t, "Item 1 Edited Title", editContents[0].Title)
	assert.Equal(t, "Description 1 Edited Content", editContents[0].Description)

	// Verify DB is updated for item1
	hashID := model.GenHashID(source.Link, "item-1")
	storedContent, err := appCore.GetContent(ctx, hashID)
	assert.NoError(t, err)
	assert.Equal(t, "Item 1 Edited Title", storedContent.Title)
	assert.Equal(t, "Description 1 Edited Content", storedContent.Description)
}



