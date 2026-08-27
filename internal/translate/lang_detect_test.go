package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameLanguage(t *testing.T) {
	tests := []struct {
		text       string
		targetLang string
		expected   bool
	}{
		{"你好世界", "zh", true},
		{"这是一篇中文博客文章测试", "zh-CN", true},
		{"Hello world, this is a test.", "zh", false},
		{"Go 1.22 正式发布，新特性解析", "zh", true},
		{"Hello world, this is a test.", "en", true},
		{"This is English", "en-US", true},
		{"你好世界", "en", false},
		{"こんにちは世界", "ja", true},
		{"Hello world", "ja", false},
		{"안녕하세요 세계", "ko", true},
		{"Hello world", "ko", false},
		{"Привет мир", "ru", true},
		{"Hello world", "ru", false},
		{"", "zh", true},
		{"   ", "en", true},
	}

	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.targetLang, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSameLanguage(tt.text, tt.targetLang))
		})
	}
}
