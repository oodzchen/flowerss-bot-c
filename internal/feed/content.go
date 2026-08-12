package feed

import (
	"strings"

	"github.com/mmcdole/gofeed"
)

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
