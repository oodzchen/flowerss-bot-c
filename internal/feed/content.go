package feed

import (
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/indes/flowerss-bot/internal/translate"
)

// ItemGUID returns the best available unique identifier for a feed item.
// It prioritizes item.GUID, falling back to item.Link, item.Title, and ItemDescription.
func ItemGUID(item *gofeed.Item) string {
	if item == nil {
		return ""
	}
	if guid := strings.TrimSpace(item.GUID); guid != "" {
		return guid
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		return link
	}
	if title := strings.TrimSpace(item.Title); title != "" {
		return title
	}
	return ItemDescription(item)
}

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

// ItemLanguage extracts the declared language of a feed item or its parent feed.
// It checks Dublin Core (<dc:language>), XML lang attributes, extensions,
// embedded HTML/XML lang attributes, and parent feed language.
func ItemLanguage(feed *gofeed.Feed, item *gofeed.Item) string {
	if item != nil {
		// 1. Dublin Core language on item (<dc:language>)
		if item.DublinCoreExt != nil && len(item.DublinCoreExt.Language) > 0 {
			for _, lang := range item.DublinCoreExt.Language {
				if lang = translate.NormalizeLangTag(lang); lang != "" {
					return lang
				}
			}
		}

		// 2. Namespaced XML / Atom extension
		if lang := extensionValue(item, "dc", "language"); lang != "" {
			return translate.NormalizeLangTag(lang)
		}
		if lang := extensionValue(item, "xml", "lang"); lang != "" {
			return translate.NormalizeLangTag(lang)
		}
		if lang := extensionValue(item, "atom", "lang"); lang != "" {
			return translate.NormalizeLangTag(lang)
		}

		// 3. Custom map
		if item.Custom != nil {
			if lang := item.Custom["language"]; lang != "" {
				return translate.NormalizeLangTag(lang)
			}
			if lang := item.Custom["lang"]; lang != "" {
				return translate.NormalizeLangTag(lang)
			}
			if lang := item.Custom["xml:lang"]; lang != "" {
				return translate.NormalizeLangTag(lang)
			}
		}

		// 4. Embedded HTML / XML lang attribute in Description or Content
		if lang := translate.ExtractXMLLanguage(item.Description); lang != "" {
			return lang
		}
		if lang := translate.ExtractXMLLanguage(item.Content); lang != "" {
			return lang
		}
	}

	// 5. Feed level language (<channel><language>, <feed xml:lang="...">, "language": "...")
	if feed != nil {
		if lang := translate.NormalizeLangTag(feed.Language); lang != "" {
			return lang
		}
		if feed.DublinCoreExt != nil && len(feed.DublinCoreExt.Language) > 0 {
			for _, lang := range feed.DublinCoreExt.Language {
				if lang = translate.NormalizeLangTag(lang); lang != "" {
					return lang
				}
			}
		}
		if feed.Custom != nil {
			if lang := feed.Custom["language"]; lang != "" {
				return translate.NormalizeLangTag(lang)
			}
			if lang := feed.Custom["lang"]; lang != "" {
				return translate.NormalizeLangTag(lang)
			}
		}
	}

	return ""
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
