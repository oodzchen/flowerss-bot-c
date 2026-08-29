package preview

import (
	"html"
	"regexp"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
)

var (
	scriptPattern    = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`)
	stylePattern     = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style\s*>`)
	lineBreakPattern = regexp.MustCompile(`(?i)<br\s*/?>`)
	blockTagPattern  = regexp.MustCompile(`(?i)</?(?:address|article|aside|blockquote|div|footer|h[1-6]|header|hr|li|main|nav|ol|p|pre|section|table|tr|ul)\b[^>]*>`)
)

func TrimDescription(desc string, limit int) string {
	if limit == 0 {
		return ""
	}

	// Feed bodies are commonly HTML, escaped HTML, or plain text. Remove
	// non-visible content, preserve common block boundaries, then collapse noisy
	// indentation without joining separate paragraphs together.
	desc = html.UnescapeString(desc)
	desc = scriptPattern.ReplaceAllString(desc, "")
	desc = stylePattern.ReplaceAllString(desc, "")
	desc = lineBreakPattern.ReplaceAllString(desc, "\n")
	desc = blockTagPattern.ReplaceAllString(desc, "\n")
	desc = html.UnescapeString(strip.StripTags(desc))
	desc = strings.ReplaceAll(desc, "\u00a0", " ")

	lines := strings.Split(strings.ReplaceAll(desc, "\r", "\n"), "\n")
	cleanLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	desc = strings.Join(cleanLines, "\n")

	contentDescRune := []rune(desc)
	if limit > 0 {
		if len(contentDescRune) > limit {
			if limit == 1 {
				return "…"
			}
			desc = string(contentDescRune[:limit-1]) + "…"
		}
	} else {
		absLimit := -limit
		if len(contentDescRune) > absLimit {
			if absLimit == 1 {
				return "…"
			}
			desc = "…" + string(contentDescRune[len(contentDescRune)-(absLimit-1):])
		}
	}

	return desc
}
