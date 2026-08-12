package feed

import (
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
	ext "github.com/mmcdole/gofeed/extensions"
	"github.com/stretchr/testify/assert"
)

func TestItemDescriptionAcrossFeedFormats(t *testing.T) {
	tests := []struct {
		name string
		feed string
		want string
	}{
		{
			name: "RSS description",
			feed: `<?xml version="1.0"?><rss version="2.0"><channel><title>Feed</title><link>https://example.com</link><description>Feed</description><item><title>Post</title><link>https://example.com/post</link><description><![CDATA[<p>RSS summary</p>]]></description></item></channel></rss>`,
			want: "<p>RSS summary</p>",
		},
		{
			name: "RSS content encoded",
			feed: `<?xml version="1.0"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>Feed</title><link>https://example.com</link><description>Feed</description><item><title>Post</title><link>https://example.com/post</link><content:encoded><![CDATA[<p>RSS full body</p>]]></content:encoded></item></channel></rss>`,
			want: "<p>RSS full body</p>",
		},
		{
			name: "Atom summary",
			feed: `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Feed</title><id>feed</id><updated>2026-01-01T00:00:00Z</updated><entry><title>Post</title><id>post</id><updated>2026-01-01T00:00:00Z</updated><summary type="html">Atom summary</summary></entry></feed>`,
			want: "Atom summary",
		},
		{
			name: "Atom content",
			feed: `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Feed</title><id>feed</id><updated>2026-01-01T00:00:00Z</updated><entry><title>Post</title><id>post</id><updated>2026-01-01T00:00:00Z</updated><content type="html">Atom full body</content></entry></feed>`,
			want: "Atom full body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := gofeed.NewParser().Parse(strings.NewReader(tt.feed))
			assert.NoError(t, err)
			if assert.Len(t, parsed.Items, 1) {
				assert.Equal(t, tt.want, ItemDescription(parsed.Items[0]))
			}
		})
	}
}

func TestItemDescription(t *testing.T) {
	t.Run("prefers feed summary", func(t *testing.T) {
		item := &gofeed.Item{Description: "short summary", Content: "full content"}
		assert.Equal(t, "short summary", ItemDescription(item))
	})

	t.Run("falls back to full content", func(t *testing.T) {
		item := &gofeed.Item{Content: "full content"}
		assert.Equal(t, "full content", ItemDescription(item))
	})

	t.Run("supports iTunes summary", func(t *testing.T) {
		item := &gofeed.Item{ITunesExt: &ext.ITunesItemExtension{Summary: "episode summary"}}
		assert.Equal(t, "episode summary", ItemDescription(item))
	})

	t.Run("supports media description", func(t *testing.T) {
		item := &gofeed.Item{Extensions: ext.Extensions{
			"media": {"description": {{Value: "media summary"}}},
		}}
		assert.Equal(t, "media summary", ItemDescription(item))
	})

	t.Run("handles nil item", func(t *testing.T) {
		assert.Empty(t, ItemDescription(nil))
	})
}

func TestItemContent(t *testing.T) {
	t.Run("prefers full content", func(t *testing.T) {
		item := &gofeed.Item{Description: "short summary", Content: "full content"}
		assert.Equal(t, "full content", ItemContent(item))
	})

	t.Run("falls back to feed summary", func(t *testing.T) {
		item := &gofeed.Item{Description: "short summary"}
		assert.Equal(t, "short summary", ItemContent(item))
	})
}
