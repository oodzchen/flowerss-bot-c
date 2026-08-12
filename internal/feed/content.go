package feed

import (
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// ItemPublishedAt returns the publication time normalized by gofeed. Some
// Atom/JSON feeds omit a distinct publication time, so updated is used as a
// fallback.
func ItemPublishedAt(item *gofeed.Item) *time.Time {
	if item == nil {
		return nil
	}
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}
	return item.UpdatedParsed
}

// SortItemsOldestFirst returns a sorted copy without changing the parser's
// item slice. Undated entries are kept in their original order before dated
// entries because their chronological position cannot be determined.
func SortItemsOldestFirst(items []*gofeed.Item) []*gofeed.Item {
	sorted := append([]*gofeed.Item(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := ItemPublishedAt(sorted[i])
		right := ItemPublishedAt(sorted[j])
		switch {
		case left == nil && right == nil:
			return false
		case left == nil:
			return true
		case right == nil:
			return false
		default:
			return left.Before(*right)
		}
	})
	return sorted
}

// ItemDescription returns the best summary-like value exposed by gofeed.
// RSS <description>, Atom <summary>, and JSON Feed summary are normalized into
// Description. Full content is used only when a summary is unavailable.
func ItemDescription(item *gofeed.Item) string {
	if item == nil {
		return ""
	}
	if strings.TrimSpace(item.Description) != "" {
		return item.Description
	}
	if item.ITunesExt != nil && strings.TrimSpace(item.ITunesExt.Summary) != "" {
		return item.ITunesExt.Summary
	}
	if description := extensionValue(item, "media", "description"); description != "" {
		return description
	}
	return item.Content
}

// ItemContent returns the fullest article body available. Falling back to the
// description also lets description-only RSS feeds use Telegraph previews.
func ItemContent(item *gofeed.Item) string {
	if item == nil {
		return ""
	}
	if strings.TrimSpace(item.Content) != "" {
		return item.Content
	}
	return ItemDescription(item)
}

func extensionValue(item *gofeed.Item, namespace, name string) string {
	if item.Extensions == nil {
		return ""
	}
	namespaced, ok := item.Extensions[namespace]
	if !ok {
		return ""
	}
	values := namespaced[name]
	for _, value := range values {
		if strings.TrimSpace(value.Value) != "" {
			return value.Value
		}
	}
	return ""
}
