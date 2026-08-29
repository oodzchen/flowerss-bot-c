package preview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimDescription(t *testing.T) {
	tests := []struct {
		name  string
		desc  string
		limit int
		want  string
	}{
		{name: "disabled", desc: "text", limit: 0, want: ""},
		{name: "plain text", desc: "  hello   world  ", limit: 100, want: "hello world"},
		{name: "escaped html", desc: "&lt;p&gt;first&lt;/p&gt;&lt;p&gt;second&lt;/p&gt;", limit: 100, want: "first\nsecond"},
		{name: "html blocks", desc: "<p>Hello&nbsp;world</p><BR/>next", limit: 100, want: "Hello world\nnext"},
		{name: "unsafe content", desc: "before<script>alert(1)</script><style>.x{}</style>after", limit: 100, want: "beforeafter"},
		{name: "rune limit positive", desc: "一二三四五", limit: 4, want: "一二三…"},
		{name: "rune limit positive exact", desc: "一二三四五", limit: 5, want: "一二三四五"},
		{name: "rune limit positive 1", desc: "一二三四五", limit: 1, want: "…"},
		{name: "rune limit negative", desc: "一二三四五", limit: -4, want: "…三四五"},
		{name: "rune limit negative exact", desc: "一二三四五", limit: -5, want: "一二三四五"},
		{name: "rune limit negative exceeding", desc: "一二三四五", limit: -10, want: "一二三四五"},
		{name: "rune limit negative 1", desc: "一二三四五", limit: -1, want: "…"},
		{name: "rune limit negative 400 chars tail", desc: "abcdefghijklmnopqrstuvwxyz", limit: -5, want: "…wxyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TrimDescription(tt.desc, tt.limit))
		})
	}
}
